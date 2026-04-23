package events

import (
	"context"
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
	entevent "encore.app/db/ent/event"
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
	if strings.TrimSpace(req.HostID) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("host_id is required").Err()
	}
	if req.EventDate.Before(time.Now().Add(-5 * time.Minute)) {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("event_date cannot be in the past").Err()
	}

	ev, err := insertEvent(ctx, ad, req)
	if err != nil {
		return nil, err
	}

	return &GetEventResponse{Event: *ev}, nil
}

// ListEvents returns events scoped to the caller's company.
// SA/ADM/HR see all statuses; EMP sees only ACTIVE.
//
//encore:api auth method=GET path=/events
func ListEvents(ctx context.Context) (*ListEventsResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}

	var rows []Event
	switch ad.Role {
	case authhandler.RoleSA, authhandler.RoleADM, authhandler.RoleHR:
		rows, err = queryAllEvents(ctx, ad.CompanyID)
	default:
		rows, err = queryActiveEvents(ctx, ad.CompanyID)
	}
	if err != nil {
		return nil, err
	}

	return &ListEventsResponse{Events: rows, Total: len(rows)}, nil
}

// GetEvent returns a single event. Employees can only see ACTIVE events;
// SA/ADM/HR see any status.
//
//encore:api auth method=GET path=/events/:id
func GetEvent(ctx context.Context, id string) (*GetEventResponse, error) {
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}

	ev, err := queryEventByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !canSeeEvent(ad, ev) {
		return nil, errs.B().Code(errs.NotFound).Msg("event not found").Err()
	}

	return &GetEventResponse{Event: *ev}, nil
}

// UpdateEvent partially updates an event. Allowed roles: SA, ADM, HR.
//
//encore:api auth method=PATCH path=/events/:id
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

	ev, err := updateEvent(ctx, id, req)
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

func insertEvent(ctx context.Context, ad *authhandler.AuthData, req *CreateEventRequest) (*Event, error) {
	clientID, err := uuid.Parse(ad.CompanyID)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("missing or invalid company in auth context").Err()
	}
	hostID, err := uuid.Parse(req.HostID)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid host_id format").Err()
	}

	hostExists, err := Client.User.
		Query().
		Where(user.IDEQ(hostID)).
		Exist(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to verify host").Cause(err).Err()
	}
	if !hostExists {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("host user not found").Err()
	}

	builder := Client.Event.
		Create().
		SetClientID(clientID).
		SetHostID(hostID).
		SetTitle(strings.TrimSpace(req.Title)).
		SetEventDate(req.EventDate).
		SetStatus(entevent.StatusACTIVE)

	if req.ZoomLink != nil {
		builder = builder.SetZoomLink(*req.ZoomLink)
	}
	if req.Description != nil {
		builder = builder.SetDescription(*req.Description)
	}

	row, err := builder.Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid client or host reference").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to create event").Cause(err).Err()
	}

	return entToEvent(row), nil
}

func queryAllEvents(ctx context.Context, companyID string) ([]Event, error) {
	q := Client.Event.Query()
	if cid, err := uuid.Parse(companyID); err == nil {
		q = q.Where(entevent.ClientIDEQ(cid))
	}

	rows, err := q.Order(ent.Desc(entevent.FieldEventDate)).All(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to list events").Cause(err).Err()
	}

	out := make([]Event, 0, len(rows))
	for _, r := range rows {
		out = append(out, *entToEvent(r))
	}
	return out, nil
}

func queryActiveEvents(ctx context.Context, companyID string) ([]Event, error) {
	q := Client.Event.
		Query().
		Where(entevent.StatusEQ(entevent.StatusACTIVE))

	if cid, err := uuid.Parse(companyID); err == nil {
		q = q.Where(entevent.ClientIDEQ(cid))
	}

	rows, err := q.Order(ent.Asc(entevent.FieldEventDate)).All(ctx)
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to list active events").Cause(err).Err()
	}

	out := make([]Event, 0, len(rows))
	for _, r := range rows {
		out = append(out, *entToEvent(r))
	}
	return out, nil
}

func queryEventByID(ctx context.Context, id string) (*Event, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	row, err := Client.Event.
		Query().
		Where(entevent.IDEQ(uid)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("event not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to get event").Cause(err).Err()
	}

	return entToEvent(row), nil
}

func updateEvent(ctx context.Context, id string, req *UpdateEventRequest) (*Event, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid id format").Err()
	}

	builder := Client.Event.UpdateOneID(uid)

	if req.Title != nil {
		builder = builder.SetTitle(strings.TrimSpace(*req.Title))
	}
	if req.EventDate != nil {
		builder = builder.SetEventDate(*req.EventDate)
	}
	if req.HostID != nil {
		hostID, err := uuid.Parse(*req.HostID)
		if err != nil {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("invalid host_id format").Err()
		}
		hostExists, err := Client.User.Query().Where(user.IDEQ(hostID)).Exist(ctx)
		if err != nil {
			return nil, errs.B().Code(errs.Internal).Msg("failed to verify host").Cause(err).Err()
		}
		if !hostExists {
			return nil, errs.B().Code(errs.InvalidArgument).Msg("host user not found").Err()
		}
		builder = builder.SetHostID(hostID)
	}
	if req.ZoomLink != nil {
		builder = builder.SetZoomLink(*req.ZoomLink)
	}
	if req.Description != nil {
		builder = builder.SetDescription(*req.Description)
	}
	if req.Status != nil {
		builder = builder.SetStatus(entevent.Status(*req.Status))
	}

	row, err := builder.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("event not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to update event").Cause(err).Err()
	}

	return entToEvent(row), nil
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

	row, err := builder.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("event not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to update materials").Cause(err).Err()
	}

	return entToEvent(row), nil
}

// ════ HELPERS ════

func getAuthData() (*authhandler.AuthData, error) {
	ad, ok := auth.Data().(*authhandler.AuthData)
	if !ok {
		return nil, errs.B().Code(errs.Unauthenticated).Msg("not authenticated").Err()
	}
	return ad, nil
}

func requireRole(ad *authhandler.AuthData, allowed ...authhandler.UserRole) error {
	for _, r := range allowed {
		if ad.Role == r {
			return nil
		}
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

func entToEvent(e *ent.Event) *Event {
	return &Event{
		ID:           e.ID.String(),
		ClientID:     e.ClientID.String(),
		HostID:       e.HostID.String(),
		Title:        e.Title,
		Description:  e.Description,
		ZoomLink:     e.ZoomLink,
		EventDate:    e.EventDate,
		MaterialsURL: e.MaterialsURL,
		Status:       EventStatus(e.Status),
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}
