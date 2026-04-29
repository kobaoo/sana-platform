package scorm_progress

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"encore.dev/beta/errs"
	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"

	"encore.app/auth/authhandler"
	"encore.app/db/ent"
	"encore.app/db/ent/scormcourse"
	"encore.app/db/ent/scormprogress"
)

type courseCertificateData struct {
	progress *ent.ScormProgress
	employee *ent.Employee
	course   *ent.ScormCourse
}

//encore:api auth raw method=GET path=/course-progress/:id/certificate
func DownloadCourseCertificate(w http.ResponseWriter, r *http.Request) {
	handleDownloadCourseCertificate(w, r)
}

func handleDownloadCourseCertificate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ad, err := getAuthData()
	if err != nil {
		http.Error(w, "not authenticated", http.StatusUnauthorized)
		return
	}

	progressID, err := parseProgressIDFromPath(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pdfBytes, err := generateAccessibleCourseCertificate(ctx, ad, progressID)
	if err != nil {
		writeCertificateError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="certificate_%s.pdf"`, progressID))
	_, _ = w.Write(pdfBytes)
}

func GenerateCourseCertificate(ctx context.Context, progressID uuid.UUID) ([]byte, error) {
	data, err := loadCourseCertificateData(ctx, progressID)
	if err != nil {
		return nil, err
	}
	if data.progress.Status != scormprogress.StatusCOMPLETED {
		return nil, errs.B().Code(errs.FailedPrecondition).Msg("course progress is not completed").Err()
	}
	if data.progress.CompletedAt == nil {
		return nil, errs.B().Code(errs.FailedPrecondition).Msg("course progress has no completion date").Err()
	}

	return renderCourseCertificatePDF(data, time.Now().UTC())
}

func generateAccessibleCourseCertificate(ctx context.Context, ad *authhandler.AuthData, progressID uuid.UUID) ([]byte, error) {
	if _, _, err := getAccessibleProgress(ctx, ad, progressID); err != nil {
		return nil, err
	}
	return GenerateCourseCertificate(ctx, progressID)
}

func loadCourseCertificateData(ctx context.Context, progressID uuid.UUID) (*courseCertificateData, error) {
	progress, err := Client.ScormProgress.
		Query().
		Where(scormprogress.IDEQ(progressID)).
		WithProgress().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errs.B().Code(errs.NotFound).Msg("course progress not found").Err()
		}
		return nil, errs.B().Code(errs.Internal).Msg("failed to get course progress").Cause(err).Err()
	}

	employeeRow, err := getEmployeeByID(ctx, progress.EmployeeID)
	if err != nil {
		if errs.Code(err) == errs.NotFound {
			return nil, errs.B().Code(errs.NotFound).Msg("course progress not found").Err()
		}
		return nil, err
	}

	courseRow := progress.Edges.Progress
	if courseRow == nil {
		courseRow, err = Client.ScormCourse.
			Query().
			Where(scormcourse.ID(progress.CourseID)).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, errs.B().Code(errs.NotFound).Msg("course not found").Err()
			}
			return nil, errs.B().Code(errs.Internal).Msg("failed to get course").Cause(err).Err()
		}
	}

	return &courseCertificateData{
		progress: progress,
		employee: employeeRow,
		course:   courseRow,
	}, nil
}

func renderCourseCertificatePDF(data *courseCertificateData, generatedAt time.Time) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Certificate of Completion", true)
	pdf.SetAuthor("Sana Platform", true)
	pdf.SetCompression(false)
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 24)
	pdf.CellFormat(0, 16, "Certificate of Completion", "", 1, "C", false, 0, "")
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 14)
	certText := fmt.Sprintf(
		"This certifies that %s has successfully completed the course %s.",
		data.employee.FullName,
		data.course.Title,
	)
	pdf.MultiCell(0, 9, certText, "", "C", false)
	pdf.Ln(6)

	if data.course.Description != nil && strings.TrimSpace(*data.course.Description) != "" {
		pdf.SetFont("Arial", "B", 13)
		pdf.CellFormat(0, 8, "Course Description", "", 1, "L", false, 0, "")
		pdf.SetFont("Arial", "", 12)
		pdf.MultiCell(0, 7, *data.course.Description, "", "L", false)
		pdf.Ln(4)
	}

	pdf.SetFont("Arial", "", 12)
	pdf.CellFormat(0, 8, fmt.Sprintf("Completion Date: %s", data.progress.CompletedAt.UTC().Format("02 Jan 2006")), "", 1, "L", false, 0, "")
	if data.progress.Score != nil {
		pdf.CellFormat(0, 8, fmt.Sprintf("Score: %d", *data.progress.Score), "", 1, "L", false, 0, "")
	}
	pdf.CellFormat(0, 8, fmt.Sprintf("Generated Date: %s", generatedAt.UTC().Format("02 Jan 2006")), "", 1, "L", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, errs.B().Code(errs.Internal).Msg("failed to generate certificate pdf").Cause(err).Err()
	}
	return buf.Bytes(), nil
}

func parseProgressIDFromPath(path string) (uuid.UUID, error) {
	for _, seg := range strings.Split(path, "/") {
		if id, err := uuid.Parse(seg); err == nil {
			return id, nil
		}
	}
	return uuid.Nil, errs.B().Code(errs.InvalidArgument).Msg("path parameter 'id' is required").Err()
}

func writeCertificateError(w http.ResponseWriter, err error) {
	switch errs.Code(err) {
	case errs.InvalidArgument:
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errs.Unauthenticated:
		http.Error(w, err.Error(), http.StatusUnauthorized)
	case errs.PermissionDenied:
		http.Error(w, err.Error(), http.StatusForbidden)
	case errs.NotFound:
		http.Error(w, err.Error(), http.StatusNotFound)
	case errs.FailedPrecondition:
		http.Error(w, err.Error(), http.StatusPreconditionFailed)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
