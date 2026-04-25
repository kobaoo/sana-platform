package events

import (
	"context"
	"slices"
	"strings"
	"time"

	"encore.app/auth/authhandler"
	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"encore.dev/storage/sqldb"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"encore.app/db/ent"
	"encore.app/db/ent/employee"
	entevent "encore.app/db/ent/event"
	"encore.app/db/ent/eventparticipant"
	"encore.app/db/ent/user"
)

// ════ DATABASE ════

var (
	db     = sqldb.Named("lms")
	Client = newEntClient()
)

func newEntClient() *ent.Client {
	drv := entsql.OpenDB(dialect.Postgres, db.Stdlib())
	return ent.NewClient(ent.Driver(drv))
}

// ════ ENDPOINTS ════

// CreateEvent creates a new event with status=ACTIVE (immediately published).
// If employee_ids are provided, they are pre-enrolled into the event inside the
// same transaction (principal attendees); remaining seats stay open for
// self-enrollment by other employees of the same client.
// Allowed roles: SA, ADM, HR.
//
//encore:api auth method=POST path=/events
func CreateEvent(ctx context.Context, req *CreateEventRequest) (*GetEventResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if err := requireRole(ad, authhandler.RoleSA, authhandler.RoleADM, authhandler.RoleHR); err != nil {
		return nil, err
	}

	if strings.TrimSpace(req.Title) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("title is required").Err()
	}
	if req.EventDate.IsZero() {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("event_date is required").Err()
	}
	if req.EventDate.Before(time.Now().Add(-5 * time.Minute)) {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("event_date cannot be in the past").Err()
	}
	if strings.TrimSpace(req.ZoomLink) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("zoom_link is required").Err()
	}
	if req.MaxParticipants <= 0 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("max_participants must be greater than zero").Err()
	}

	employeeIDs, err := dedupeUUIDs(req.EmployeeIDs)
	if err != nil {
		return nil, err
	}
	if len(employeeIDs) > req.MaxParticipants {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("employee_ids exceed max_participants").Err()
	}

	hostID, err := resolveHostID(ctx, ad, req.HostID)
	if err != nil {
		return nil, err
	}

	ev, err := insertEvent(ctx, ad, hostID, employeeIDs, req)
	if err != nil {
		return nil, err
	}

	return &GetEventResponse{Event: *ev}, nil
}

// ListEvents returns events scoped to the caller's company.
// SA/ADM/HR see all statuses; EMP sees only ACTIVE.
// Optional filters: from, to (YYYY-MM-DD or RFC3339), status.
//
//encore:api auth method=GET path=/events
func ListEvents(ctx context.Context, params *ListEventsParams) (*ListEventsResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}

	from, to, err := parseDateRange(params.From, params.To)
	if err != nil {
		return nil, err
	}

	statusFilter, err := parseStatus(params.Status)
	if err != nil {
		return nil, err
	}

	// Employees are restricted to ACTIVE events regardless of the status filter.
	if ad.Role == authhandler.RoleEMP {
		s := entevent.StatusACTIVE
		statusFilter = &s
	}

	rows, err := queryEvents(ctx, ad.CompanyID, from, to, statusFilter)
	if err != nil {
		return nil, err
	}

	if ad.Role == authhandler.RoleEMP {
		empID, lookupErr := lookupEmployeeID(ctx, ad)
		switch {
		case lookupErr == nil:
			if err := annotateRegistrations(ctx, rows, empID); err != nil {
				return nil, err
			}
		case errs.Code(lookupErr) == errs.FailedPrecondition:
			// caller has no employee profile — leave is_registered=false
		default:
			return nil, lookupErr
		}
		redactZoomForUnregistered(rows)
	}

	return &ListEventsResponse{Events: rows, Total: len(rows)}, nil
}

// GetEvent returns a single event together with its participants.
// Employees can only see ACTIVE events; SA/ADM/HR see any status.
// For EMP, the response carries `is_registered` and the zoom_link is
// hidden until the caller is enrolled.
//
//encore:api auth method=GET path=/events/:id
func GetEvent(ctx context.Context, id string) (*GetEventResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}

	ev, err := queryEventByID(ctx, id, true)
	if err != nil {
		return nil, err
	}

	if !canSeeEvent(ad, ev) {
		return nil, errs.B().Code(errs.NotFound).Msg("event not found").Err()
	}

	if ad.Role == authhandler.RoleEMP {
		// Hide participant details from regular employees.
		ev.Participants = nil

		empID, lookupErr := lookupEmployeeID(ctx, ad)
		switch {
		case lookupErr == nil:
			registered, err := isRegistered(ctx, uuid.MustParse(ev.ID), empID)
			if err != nil {
				return nil, err
			}
			ev.IsRegistered = registered
		case errs.Code(lookupErr) == errs.FailedPrecondition:
			// caller has no employee profile — treat as not registered
		default:
			return nil, lookupErr
		}
		if !ev.IsRegistered {
			ev.ZoomLink = ""
		}
	}

	return &GetEventResponse{Event: *ev}, nil
}

// UpdateEvent updates an event (partial update; only provided fields change).
// Passing employee_ids replaces the pre-enrolled list entirely.
// Allowed roles: SA, ADM, HR.
//
//encore:api auth method=PUT path=/events/:id
func UpdateEvent(ctx context.Context, id string, req *UpdateEventRequest) (*GetEventResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if err := requireRole(ad, authhandler.RoleSA, authhandler.RoleADM, authhandler.RoleHR); err != nil {
		return nil, err
	}

	if req.Status != nil && !req.Status.IsValid() {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid status").Err()
	}
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("title cannot be empty").Err()
	}
	if req.ZoomLink != nil && strings.TrimSpace(*req.ZoomLink) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("zoom_link cannot be empty").Err()
	}
	if req.MaxParticipants != nil && *req.MaxParticipants <= 0 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("max_participants must be greater than zero").Err()
	}

	var replacementIDs []uuid.UUID
	if req.EmployeeIDs != nil {
		replacementIDs, err = dedupeUUIDs(*req.EmployeeIDs)
		if err != nil {
			return nil, err
		}
	}

	ev, err := updateEvent(ctx, ad, id, req, replacementIDs)
	if err != nil {
		return nil, err
	}

	return &GetEventResponse{Event: *ev}, nil
}

// DeleteEvent soft-deletes an event by setting status=CANCELLED.
// Allowed roles: SA, ADM, HR.
//
//encore:api auth method=DELETE path=/events/:id
func DeleteEvent(ctx context.Context, id string) (*DeleteEventResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if err := requireRole(ad, authhandler.RoleSA, authhandler.RoleADM, authhandler.RoleHR); err != nil {
		return nil, err
	}

	if err := softDeleteEvent(ctx, id); err != nil {
		return nil, err
	}

	return &DeleteEventResponse{Message: "event cancelled"}, nil
}

// RegisterForEvent enrolls the calling employee into an event.
// Capacity is enforced inside a transaction: if the event is at max
// after the insert, the transaction is rolled back and FailedPrecondition
// is returned. Idempotent on the unique (event_id, employee_id) constraint —
// second register by the same employee returns AlreadyExists.
// Allowed role: EMP.
//
//encore:api auth method=POST path=/events/:id/register
func RegisterForEvent(ctx context.Context, id string) (*GetEventResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if err := requireRole(ad, authhandler.RoleEMP); err != nil {
		return nil, err
	}

	eventID, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid event id").Err()
	}

	empID, err := lookupEmployeeID(ctx, ad)
	if err != nil {
		return nil, err
	}

	ev, err := queryEventByID(ctx, id, false)
	if err != nil {
		return nil, err
	}
	if !canSeeEvent(ad, ev) {
		return nil, errs.B().Code(errs.NotFound).Msg("event not found").Err()
	}
	if ev.Status != StatusActive {
		return nil, errs.B().Code(errs.FailedPrecondition).Msg("event is not active").Err()
	}

	if err := enrollEmployee(ctx, eventID, empID); err != nil {
		return nil, err
	}

	out, err := loadEventByID(ctx, eventID, true)
	if err != nil {
		return nil, err
	}
	out.Participants = nil
	out.IsRegistered = true
	return &GetEventResponse{Event: *out}, nil
}

// UnregisterFromEvent removes the calling employee's enrollment.
// Idempotent in spirit but returns NotFound if no enrollment exists.
// Allowed role: EMP.
//
//encore:api auth method=DELETE path=/events/:id/register
func UnregisterFromEvent(ctx context.Context, id string) (*GetEventResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if err := requireRole(ad, authhandler.RoleEMP); err != nil {
		return nil, err
	}

	eventID, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid event id").Err()
	}

	empID, err := lookupEmployeeID(ctx, ad)
	if err != nil {
		return nil, err
	}

	ev, err := queryEventByID(ctx, id, false)
	if err != nil {
		return nil, err
	}
	if !canSeeEvent(ad, ev) {
		return nil, errs.B().Code(errs.NotFound).Msg("event not found").Err()
	}

	deleted, err := Client.EventParticipant.
		Delete().
		Where(
			eventparticipant.EventIDEQ(eventID),
			eventparticipant.EmployeeIDEQ(empID),
		).
		Exec(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to unregister").Cause(err).Err()
	}
	if deleted == 0 {
		return nil, errs.B().Code(errs.NotFound).Msg("you are not registered for this event").Err()
	}

	out, err := loadEventByID(ctx, eventID, true)
	if err != nil {
		return nil, err
	}
	out.Participants = nil
	out.IsRegistered = false
	out.ZoomLink = ""
	return &GetEventResponse{Event: *out}, nil
}

// ListHosts returns potential hosts (SA/ADM/HR users) scoped to the caller's
// company. Used by the event creation form on the frontend. Supports `search`
// (case-insensitive substring match on email/full_name) and limit/offset
// pagination for lazy-load.
// Allowed roles: SA, ADM, HR.
//
//encore:api auth method=GET path=/event-hosts
func ListHosts(ctx context.Context, params *ListHostsParams) (*ListHostsResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if err := requireRole(ad, authhandler.RoleSA, authhandler.RoleADM, authhandler.RoleHR); err != nil {
		return nil, err
	}

	limit, offset := normalizeHostsPagination(params.Limit, params.Offset)
	search := strings.TrimSpace(params.Search)

	hosts, total, err := queryHosts(ctx, ad.CompanyID, search, limit, offset)
	if err != nil {
		return nil, err
	}

	return &ListHostsResponse{
		Hosts:   hosts,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+len(hosts) < total,
	}, nil
}

// SetEventMaterials attaches or replaces the materials URL
// (link to a Google Drive / OneDrive / Dropbox folder).
// Allowed roles: SA, ADM, HR.
//
//encore:api auth method=PUT path=/events/:id/materials
func SetEventMaterials(ctx context.Context, id string, req *SetMaterialsRequest) (*GetEventResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if err := requireRole(ad, authhandler.RoleSA, authhandler.RoleADM, authhandler.RoleHR); err != nil {
		return nil, err
	}

	url := strings.TrimSpace(req.MaterialsURL)
	if url == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("materials_url is required").Err()
	}

	ev, err := setMaterials(ctx, id, &url)
	if err != nil {
		return nil, err
	}

	return &GetEventResponse{Event: *ev}, nil
}

// ClearEventMaterials removes the materials URL from an event.
// Allowed roles: SA, ADM, HR.
//
//encore:api auth method=DELETE path=/events/:id/materials
func ClearEventMaterials(ctx context.Context, id string) (*GetEventResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}
	if err := requireRole(ad, authhandler.RoleSA, authhandler.RoleADM, authhandler.RoleHR); err != nil {
		return nil, err
	}

	ev, err := setMaterials(ctx, id, nil)
	if err != nil {
		return nil, err
	}

	return &GetEventResponse{Event: *ev}, nil
}

// ════ INTERNAL ════

func insertEvent(ctx context.Context, ad *authhandler.AuthData, hostID uuid.UUID, employeeIDs []uuid.UUID, req *CreateEventRequest) (*Event, error) {
	clientID, err := uuid.Parse(ad.CompanyID)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("missing or invalid company in auth context").Err()
	}

	if err := validateEmployeesInClient(ctx, employeeIDs, clientID); err != nil {
		return nil, err
	}

	tx, err := Client.Tx(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to begin transaction").Cause(err).Err()
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	builder := tx.Event.
		Create().
		SetClientID(clientID).
		SetHostID(hostID).
		SetTitle(strings.TrimSpace(req.Title)).
		SetEventDate(req.EventDate).
		SetZoomLink(strings.TrimSpace(req.ZoomLink)).
		SetMaxParticipants(req.MaxParticipants).
		SetStatus(entevent.StatusACTIVE)

	if req.Description != nil {
		builder = builder.SetDescription(*req.Description)
	}

	row, err := builder.Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			err = errs.B().Code(errs.InvalidArgument).Msg("invalid client or host reference").Err()
			return nil, err
		}
		err = errs.B().Code(errs.Internal).Msg("failed to create event").Cause(err).Err()
		return nil, err
	}

	if len(employeeIDs) > 0 {
		if err = bulkInsertParticipants(ctx, tx, row.ID, employeeIDs); err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(); err != nil {
		err = errs.B().Code(errs.Internal).Msg("failed to commit transaction").Cause(err).Err()
		return nil, err
	}

	return loadEventByID(ctx, row.ID, true)
}

func updateEvent(ctx context.Context, ad *authhandler.AuthData, id string, req *UpdateEventRequest, replacementIDs []uuid.UUID) (*Event, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	clientID, err := uuid.Parse(ad.CompanyID)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("missing or invalid company in auth context").Err()
	}

	// Ensure the event exists and is in the caller's company.
	existing, err := Client.Event.
		Query().
		Where(entevent.IDEQ(uid), entevent.ClientIDEQ(clientID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("event not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to load event").Cause(err).Err()
	}

	// Determine effective max_participants for bounds-checking the new participants list.
	effectiveMax := existing.MaxParticipants
	if req.MaxParticipants != nil {
		effectiveMax = *req.MaxParticipants
	}

	if req.EmployeeIDs != nil && len(replacementIDs) > effectiveMax {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("employee_ids exceed max_participants").Err()
	}
	if req.EmployeeIDs != nil {
		if err := validateEmployeesInClient(ctx, replacementIDs, clientID); err != nil {
			return nil, err
		}
	}

	tx, err := Client.Tx(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to begin transaction").Cause(err).Err()
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	builder := tx.Event.UpdateOneID(uid)

	if req.Title != nil {
		builder = builder.SetTitle(strings.TrimSpace(*req.Title))
	}
	if req.EventDate != nil {
		builder = builder.SetEventDate(*req.EventDate)
	}
	if req.HostID != nil {
		hostID, err := uuid.Parse(*req.HostID)
		if err != nil {
			err = errs.B().Code(errs.InvalidArgument).Msg("invalid host_id format").Err()
			return nil, err
		}
		hostExists, err2 := tx.User.Query().Where(user.IDEQ(hostID)).Exist(ctx)
		if err2 != nil {
			err = errs.B().Code(errs.Internal).Msg("failed to verify host").Cause(err2).Err()
			return nil, err
		}
		if !hostExists {
			err = errs.B().Code(errs.InvalidArgument).Msg("host user not found").Err()
			return nil, err
		}
		builder = builder.SetHostID(hostID)
	}
	if req.ZoomLink != nil {
		builder = builder.SetZoomLink(strings.TrimSpace(*req.ZoomLink))
	}
	if req.MaxParticipants != nil {
		builder = builder.SetMaxParticipants(*req.MaxParticipants)
	}
	if req.Description != nil {
		builder = builder.SetDescription(*req.Description)
	}
	if req.Status != nil {
		builder = builder.SetStatus(entevent.Status(*req.Status))
	}

	if _, err = builder.Save(ctx); err != nil {
		if ent.IsNotFound(err) {
			err = errs.B().Code(errs.NotFound).Msg("event not found").Err()
			return nil, err
		}
		err = errs.B().Code(errs.Internal).Msg("failed to update event").Cause(err).Err()
		return nil, err
	}

	if req.EmployeeIDs != nil {
		if _, err = tx.EventParticipant.
			Delete().
			Where(eventparticipant.EventIDEQ(uid)).
			Exec(ctx); err != nil {
			err = errs.B().Code(errs.Internal).Msg("failed to replace participants").Cause(err).Err()
			return nil, err
		}
		if len(replacementIDs) > 0 {
			if err = bulkInsertParticipants(ctx, tx, uid, replacementIDs); err != nil {
				return nil, err
			}
		}
	}

	if err = tx.Commit(); err != nil {
		err = errs.B().Code(errs.Internal).Msg("failed to commit transaction").Cause(err).Err()
		return nil, err
	}

	return loadEventByID(ctx, uid, true)
}

func bulkInsertParticipants(ctx context.Context, tx *ent.Tx, eventID uuid.UUID, employeeIDs []uuid.UUID) error {
	builders := make([]*ent.EventParticipantCreate, 0, len(employeeIDs))
	for _, eid := range employeeIDs {
		builders = append(builders,
			tx.EventParticipant.
				Create().
				SetEventID(eventID).
				SetEmployeeID(eid),
		)
	}
	if _, err := tx.EventParticipant.CreateBulk(builders...).Save(ctx); err != nil {
		if ent.IsConstraintError(err) {
			return errs.B().Code(errs.InvalidArgument).Msg("one of the selected employees does not exist or is already enrolled").Err()
		}
		return errs.B().Code(errs.Internal).Msg("failed to enroll participants").Cause(err).Err()
	}
	return nil
}

func validateEmployeesInClient(ctx context.Context, employeeIDs []uuid.UUID, clientID uuid.UUID) error {
	if len(employeeIDs) == 0 {
		return nil
	}
	count, err := Client.Employee.
		Query().
		Where(
			employee.IDIn(employeeIDs...),
			employee.ClientIDEQ(clientID),
			employee.IsDeletedEQ(false),
		).
		Count(ctx)
	if err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to validate employees").Cause(err).Err()
	}
	if count != len(employeeIDs) {
		return errs.B().Code(errs.InvalidArgument).Msg("one or more employee_ids are invalid or not in your client").Err()
	}
	return nil
}

func queryEvents(ctx context.Context, companyID string, from, to *time.Time, status *entevent.Status) ([]Event, error) {
	q := Client.Event.Query().WithParticipants()

	if cid, err := uuid.Parse(companyID); err == nil {
		q = q.Where(entevent.ClientIDEQ(cid))
	}
	if from != nil {
		q = q.Where(entevent.EventDateGTE(*from))
	}
	if to != nil {
		q = q.Where(entevent.EventDateLTE(*to))
	}
	if status != nil {
		q = q.Where(entevent.StatusEQ(*status))
	}

	rows, err := q.Order(ent.Asc(entevent.FieldEventDate)).All(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to list events").Cause(err).Err()
	}

	out := make([]Event, 0, len(rows))
	for _, r := range rows {
		out = append(out, *entToEvent(r, false))
	}
	return out, nil
}

func queryEventByID(ctx context.Context, id string, withParticipants bool) (*Event, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}
	return loadEventByID(ctx, uid, withParticipants)
}

func loadEventByID(ctx context.Context, uid uuid.UUID, withParticipants bool) (*Event, error) {
	q := Client.Event.Query().Where(entevent.IDEQ(uid))
	if withParticipants {
		q = q.WithParticipants(func(pq *ent.EventParticipantQuery) {
			pq.WithEmployee()
		})
	}

	row, err := q.Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("event not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to get event").Cause(err).Err()
	}

	return entToEvent(row, withParticipants), nil
}

func softDeleteEvent(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	exists, err := Client.Event.
		Query().
		Where(
			entevent.IDEQ(uid),
			entevent.StatusNEQ(entevent.StatusCANCELLED),
		).
		Exist(ctx)
	if err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to delete event").Cause(err).Err()
	}
	if !exists {
		return errs.B().Code(errs.NotFound).Msg("event not found").Err()
	}

	if err := Client.Event.
		UpdateOneID(uid).
		SetStatus(entevent.StatusCANCELLED).
		Exec(ctx); err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to delete event").Cause(err).Err()
	}

	return nil
}

func setMaterials(ctx context.Context, id string, url *string) (*Event, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	builder := Client.Event.UpdateOneID(uid)
	if url == nil {
		builder = builder.ClearMaterialsURL()
	} else {
		builder = builder.SetMaterialsURL(*url)
	}

	if _, err := builder.Save(ctx); err != nil {
		if ent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("event not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to update materials").Cause(err).Err()
	}

	return loadEventByID(ctx, uid, true)
}

// resolveHostID parses the provided host_id or falls back to the caller's
// user record (resolved via keycloak_user_id) when host_id is empty.
func resolveHostID(ctx context.Context, ad *authhandler.AuthData, rawHostID string) (uuid.UUID, error) {
	rawHostID = strings.TrimSpace(rawHostID)
	if rawHostID != "" {
		hostID, err := uuid.Parse(rawHostID)
		if err != nil {
			return uuid.Nil, errs.B().Code(errs.InvalidArgument).Msg("invalid host_id format").Err()
		}
		exists, err := Client.User.Query().Where(user.IDEQ(hostID)).Exist(ctx)
		if err != nil {
			return uuid.Nil, errs.B().Code(errs.Internal).Msg("failed to verify host").Cause(err).Err()
		}
		if !exists {
			return uuid.Nil, errs.B().Code(errs.InvalidArgument).Msg("host user not found").Err()
		}
		return hostID, nil
	}

	// Auto-fill host with the caller's own user record.
	if ad.KeycloakUserID == "" {
		return uuid.Nil, errs.B().Code(errs.InvalidArgument).Msg("host_id is required").Err()
	}
	row, err := Client.User.
		Query().
		Where(user.KeycloakUserIDEQ(ad.KeycloakUserID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return uuid.Nil, errs.B().Code(errs.InvalidArgument).Msg("host_id is required").Err()
		}
		return uuid.Nil, errs.B().Code(errs.Internal).Msg("failed to resolve host").Cause(err).Err()
	}
	return row.ID, nil
}

func dedupeUUIDs(raw []string) ([]uuid.UUID, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	seen := make(map[uuid.UUID]struct{}, len(raw))
	out := make([]uuid.UUID, 0, len(raw))
	for _, s := range raw {
		id, err := uuid.Parse(strings.TrimSpace(s))
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid employee_id format").Err()
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

func parseDateRange(fromRaw, toRaw string) (*time.Time, *time.Time, error) {
	from, err := parseFlexibleDate(fromRaw, false)
	if err != nil {
		return nil, nil, errs.B().Code(errs.InvalidArgument).Msg("invalid 'from' date, expected YYYY-MM-DD or RFC3339").Err()
	}
	to, err := parseFlexibleDate(toRaw, true)
	if err != nil {
		return nil, nil, errs.B().Code(errs.InvalidArgument).Msg("invalid 'to' date, expected YYYY-MM-DD or RFC3339").Err()
	}
	if from != nil && to != nil && to.Before(*from) {
		return nil, nil, errs.B().Code(errs.InvalidArgument).Msg("'to' must be on or after 'from'").Err()
	}
	return from, to, nil
}

// parseFlexibleDate accepts YYYY-MM-DD (treated as a local calendar day) or RFC3339.
// When endOfDay is true and only a date was supplied, the returned time is the
// last instant of that day so that a whole-day filter is inclusive.
func parseFlexibleDate(raw string, endOfDay bool) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		if endOfDay {
			eod := t.Add(24*time.Hour - time.Nanosecond)
			return &eod, nil
		}
		return &t, nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return &t, nil
	}
	return nil, errs.B().Code(errs.InvalidArgument).Err()
}

// queryHosts returns users with role SA/ADM/HR scoped to the given company,
// enriched with full_name from the linked employee record when available.
// Returns the page slice and the total matching count.
//
// Visibility rule: include both fully-active users and pending admins
// (is_active=false AND is_onboarded=false — created via RegisterAdmin,
// not yet activated on first login). Blocked users (is_active=false
// AND is_onboarded=true) are excluded.
func queryHosts(ctx context.Context, companyID, search string, limit, offset int) ([]Host, int, error) {
	q := Client.User.
		Query().
		Where(
			user.Or(
				user.IsActiveEQ(true),
				user.IsOnboardedEQ(false),
			),
			user.RoleIn(
				string(authhandler.RoleSA),
				string(authhandler.RoleADM),
				string(authhandler.RoleHR),
			),
		)

	if cid, err := uuid.Parse(companyID); err == nil {
		q = q.Where(user.ClientIDEQ(cid))
	}
	if search != "" {
		q = q.Where(user.EmailContainsFold(search))
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, errs.B().Code(errs.Internal).Msg("failed to count hosts").Cause(err).Err()
	}

	rows, err := q.Clone().
		Order(ent.Asc(user.FieldEmail)).
		Limit(limit).
		Offset(offset).
		All(ctx)
	if err != nil {
		return nil, 0, errs.B().Code(errs.Internal).Msg("failed to list hosts").Cause(err).Err()
	}

	// Enrich with full_name from the linked employee record (one extra query).
	fullNames, err := hostFullNames(ctx, rows)
	if err != nil {
		return nil, 0, err
	}

	hosts := make([]Host, 0, len(rows))
	for _, u := range rows {
		hosts = append(hosts, Host{
			ID:       u.ID.String(),
			Email:    u.Email,
			Role:     u.Role,
			FullName: fullNames[u.ID],
		})
	}
	return hosts, total, nil
}

func hostFullNames(ctx context.Context, users []*ent.User) (map[uuid.UUID]string, error) {
	if len(users) == 0 {
		return nil, nil
	}
	ids := make([]uuid.UUID, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
	}
	emps, err := Client.Employee.
		Query().
		Where(employee.UserIDIn(ids...), employee.IsDeletedEQ(false)).
		Select(employee.FieldUserID, employee.FieldFullName).
		All(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to resolve host names").Cause(err).Err()
	}
	out := make(map[uuid.UUID]string, len(emps))
	for _, e := range emps {
		if e.UserID != nil {
			out[*e.UserID] = e.FullName
		}
	}
	return out, nil
}

func normalizeHostsPagination(limit, offset int) (int, int) {
	const (
		defaultLimit = 30
		maxLimit     = 50
	)
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func parseStatus(raw string) (*entevent.Status, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	s := EventStatus(strings.ToUpper(raw))
	if !s.IsValid() {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid status").Err()
	}
	es := entevent.Status(s)
	return &es, nil
}

// ════ HELPERS ════

// lookupEmployeeID resolves the calling user to their employee record via
// keycloak_user_id → user.id → employee.user_id. Returns FailedPrecondition
// when the caller has no employee record (e.g. an admin with no employee row).
func lookupEmployeeID(ctx context.Context, ad *authhandler.AuthData) (uuid.UUID, error) {
	if ad.KeycloakUserID == "" {
		return uuid.Nil, errs.B().Code(errs.Unauthenticated).Msg("missing user identity").Err()
	}
	u, err := Client.User.
		Query().
		Where(user.KeycloakUserIDEQ(ad.KeycloakUserID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return uuid.Nil, errs.B().Code(errs.FailedPrecondition).Msg("caller has no user record").Err()
		}
		return uuid.Nil, errs.B().Code(errs.Internal).Msg("failed to resolve user").Cause(err).Err()
	}
	emp, err := Client.Employee.
		Query().
		Where(employee.UserIDEQ(u.ID), employee.IsDeletedEQ(false)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return uuid.Nil, errs.B().Code(errs.FailedPrecondition).Msg("caller has no employee profile").Err()
		}
		return uuid.Nil, errs.B().Code(errs.Internal).Msg("failed to resolve employee").Cause(err).Err()
	}
	return emp.ID, nil
}

// isRegistered checks whether the given employee has an active enrollment
// in the given event.
func isRegistered(ctx context.Context, eventID, empID uuid.UUID) (bool, error) {
	exists, err := Client.EventParticipant.
		Query().
		Where(
			eventparticipant.EventIDEQ(eventID),
			eventparticipant.EmployeeIDEQ(empID),
		).
		Exist(ctx)
	if err != nil {
		return false, errs.B().Code(errs.Internal).Msg("failed to check enrollment").Cause(err).Err()
	}
	return exists, nil
}

// annotateRegistrations populates IsRegistered for each event in the slice
// using a single batched query keyed by event id.
func annotateRegistrations(ctx context.Context, evs []Event, empID uuid.UUID) error {
	if len(evs) == 0 {
		return nil
	}
	eventIDs := make([]uuid.UUID, 0, len(evs))
	for i := range evs {
		uid, err := uuid.Parse(evs[i].ID)
		if err != nil {
			continue
		}
		eventIDs = append(eventIDs, uid)
	}
	rows, err := Client.EventParticipant.
		Query().
		Where(
			eventparticipant.EmployeeIDEQ(empID),
			eventparticipant.EventIDIn(eventIDs...),
		).
		Select(eventparticipant.FieldEventID).
		All(ctx)
	if err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to load registrations").Cause(err).Err()
	}
	registered := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		registered[r.EventID.String()] = struct{}{}
	}
	for i := range evs {
		if _, ok := registered[evs[i].ID]; ok {
			evs[i].IsRegistered = true
		}
	}
	return nil
}

// redactZoomForUnregistered hides zoom_link from EMP responses when the
// caller is not registered. Keeps the field intact for registered ones.
func redactZoomForUnregistered(evs []Event) {
	for i := range evs {
		if !evs[i].IsRegistered {
			evs[i].ZoomLink = ""
		}
	}
}

// enrollEmployee inserts a participant row inside a transaction with a
// post-insert capacity check. Two concurrent registrations that both observe
// free capacity will both insert, then both re-count and the loser rolls
// back with FailedPrecondition. The unique (event_id, employee_id) constraint
// makes a duplicate register by the same employee return AlreadyExists.
func enrollEmployee(ctx context.Context, eventID, empID uuid.UUID) error {
	tx, err := Client.Tx(ctx)
	if err != nil {
		return errs.B().Code(errs.Internal).Msg("failed to begin transaction").Cause(err).Err()
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	ev, err := tx.Event.Get(ctx, eventID)
	if err != nil {
		if ent.IsNotFound(err) {
			err = errs.B().Code(errs.NotFound).Msg("event not found").Err()
			return err
		}
		err = errs.B().Code(errs.Internal).Msg("failed to load event").Cause(err).Err()
		return err
	}

	preCount, err := tx.EventParticipant.
		Query().
		Where(eventparticipant.EventIDEQ(eventID)).
		Count(ctx)
	if err != nil {
		err = errs.B().Code(errs.Internal).Msg("failed to count participants").Cause(err).Err()
		return err
	}
	if preCount >= ev.MaxParticipants {
		err = errs.B().Code(errs.FailedPrecondition).Msg("no slots available").Err()
		return err
	}

	now := time.Now()
	_, err = tx.EventParticipant.
		Create().
		SetEventID(eventID).
		SetEmployeeID(empID).
		SetJoinedAt(now).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			err = errs.B().Code(errs.AlreadyExists).Msg("you are already registered for this event").Err()
			return err
		}
		err = errs.B().Code(errs.Internal).Msg("failed to register").Cause(err).Err()
		return err
	}

	postCount, err := tx.EventParticipant.
		Query().
		Where(eventparticipant.EventIDEQ(eventID)).
		Count(ctx)
	if err != nil {
		err = errs.B().Code(errs.Internal).Msg("failed to verify capacity").Cause(err).Err()
		return err
	}
	if postCount > ev.MaxParticipants {
		err = errs.B().Code(errs.FailedPrecondition).Msg("no slots available").Err()
		return err
	}

	if err = tx.Commit(); err != nil {
		err = errs.B().Code(errs.Internal).Msg("failed to commit registration").Cause(err).Err()
		return err
	}
	return nil
}

func getAuthData() (*authhandler.AuthData, error) {
	ad, ok := auth.Data().(*authhandler.AuthData)
	if !ok {
		return nil, errs.B().Code(errs.Unauthenticated).Msg("not authenticated").Err()
	}
	return ad, nil
}

func requireRole(ad *authhandler.AuthData, allowed ...authhandler.UserRole) error {
	if slices.Contains(allowed, ad.Role) {
		return nil
	}
	return errs.B().Code(errs.PermissionDenied).Msg("insufficient permissions").Err()
}

// canSeeEvent enforces visibility: employees only see ACTIVE events
// belonging to their own company.
func canSeeEvent(ad *authhandler.AuthData, ev *Event) bool {
	switch ad.Role {
	case authhandler.RoleSA, authhandler.RoleADM, authhandler.RoleHR:
		return true
	default:
		return ev.Status == StatusActive && ev.ClientID == ad.CompanyID
	}
}

func entToEvent(e *ent.Event, includeParticipants bool) *Event {
	ev := &Event{
		ID:              e.ID.String(),
		ClientID:        e.ClientID.String(),
		HostID:          e.HostID.String(),
		Title:           e.Title,
		Description:     e.Description,
		ZoomLink:        e.ZoomLink,
		EventDate:       e.EventDate,
		MaxParticipants: e.MaxParticipants,
		MaterialsURL:    e.MaterialsURL,
		Status:          EventStatus(e.Status),
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}

	count := len(e.Edges.Participants)
	ev.ParticipantsCount = count
	ev.AvailableSlots = max(e.MaxParticipants-count, 0)

	if includeParticipants && count > 0 {
		ev.Participants = make([]Participant, 0, count)
		for _, p := range e.Edges.Participants {
			part := Participant{
				ID:               p.ID.String(),
				EmployeeID:       p.EmployeeID.String(),
				AttendanceStatus: string(p.AttendanceStatus),
				JoinedAt:         p.CreatedAt,
			}
			if p.JoinedAt != nil {
				part.JoinedAt = *p.JoinedAt
			}
			if p.Edges.Employee != nil {
				part.FullName = p.Edges.Employee.FullName
				part.Email = p.Edges.Employee.Email
				part.Department = p.Edges.Employee.Department
				part.Position = p.Edges.Employee.Position
			}
			ev.Participants = append(ev.Participants, part)
		}
	}

	return ev
}
