// Package courses tests.
//
// This file imports encore.dev/storage/objects and encore.dev/storage/sqldb and
// cannot be run with plain go test.
// Use encore test ./orgstructure/courses/... to run these tests.
package courses

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"encore.app/auth/authhandler"
	"encore.app/db/ent"
	"encore.app/db/ent/scormcourse"
	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"github.com/google/uuid"
)

const testCompanyID = "00000000-0000-0000-0000-000000000001"

func adminCtx() context.Context {
	return auth.WithContext(
		context.Background(),
		auth.UID("courses-admin"),
		&authhandler.AuthData{
			KeycloakUserID: "courses-admin",
			Role:           authhandler.RoleADM,
			CompanyID:      testCompanyID,
		},
	)
}

func employeeCtx() context.Context {
	return auth.WithContext(
		context.Background(),
		auth.UID("courses-employee"),
		&authhandler.AuthData{
			KeycloakUserID: "courses-employee",
			Role:           authhandler.RoleEMP,
			CompanyID:      testCompanyID,
		},
	)
}

func strPtr(s string) *string {
	return &s
}

func boolPtr(v bool) *bool {
	return &v
}

func makeCreateCourseRequest() *CreateCourseRequest {
	return &CreateCourseRequest{
		Title:       "Course " + uuid.NewString(),
		CategoryIDs: []uuid.UUID{uuid.New(), uuid.New()},
		Description: strPtr("Integration test course"),
		Lecturer:    strPtr("Test Lecturer"),
		ScormURL:    "scorm/uploads/" + uuid.NewString() + ".zip",
	}
}

func mustCreateCourse(t *testing.T) *Course {
	t.Helper()

	resp, err := CreateCourse(adminCtx(), makeCreateCourseRequest())
	if err != nil {
		t.Fatalf("CreateCourse setup failed: %v", err)
	}

	return resp.Course
}

func mustGetCourseRow(t *testing.T, id uuid.UUID) *ent.ScormCourse {
	t.Helper()

	row, err := Client.ScormCourse.Query().Where(scormcourse.ID(id)).Only(context.Background())
	if err != nil {
		t.Fatalf("query course row: %v", err)
	}

	return row
}

func TestUploadSCORM_Success(t *testing.T) {
	req := &UploadSCORMRequest{
		FileName: "My Course Package.zip",
		FileData: bytes.Repeat([]byte("zip-data"), 4),
	}

	resp, err := UploadSCORM(adminCtx(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.FileName != req.FileName {
		t.Errorf("expected file_name %q, got %q", req.FileName, resp.FileName)
	}
	if resp.FileSize != len(req.FileData) {
		t.Errorf("expected file_size %d, got %d", len(req.FileData), resp.FileSize)
	}
	if resp.ScormURL != "scorm/uploads/My_Course_Package.zip" {
		t.Errorf("expected sanitized scorm_url, got %q", resp.ScormURL)
	}
	if !resp.IsValid {
		t.Error("expected upload to be valid")
	}
}

func TestUploadSCORM_InvalidExtension(t *testing.T) {
	_, err := UploadSCORM(adminCtx(), &UploadSCORMRequest{
		FileName: "course.pdf",
		FileData: []byte("abc"),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestUploadSCORM_EmployeeDenied(t *testing.T) {
	_, err := UploadSCORM(employeeCtx(), &UploadSCORMRequest{
		FileName: "course.zip",
		FileData: []byte("abc"),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.PermissionDenied {
		t.Errorf("expected PermissionDenied, got %v", errs.Code(err))
	}
}

func TestUploadCourseImage_Success(t *testing.T) {
	req := &UploadCourseImageRequest{
		FileName: "Preview Image.png",
		FileData: bytes.Repeat([]byte("png-data"), 4),
	}

	resp, err := UploadCourseImage(adminCtx(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.FileName != req.FileName {
		t.Errorf("expected file_name %q, got %q", req.FileName, resp.FileName)
	}
	if resp.FileSize != len(req.FileData) {
		t.Errorf("expected file_size %d, got %d", len(req.FileData), resp.FileSize)
	}
	if resp.ImageURL != "course-images/Preview_Image.png" {
		t.Errorf("expected sanitized image_url, got %q", resp.ImageURL)
	}
	if resp.Message != "Course image uploaded successfully" {
		t.Errorf("unexpected message %q", resp.Message)
	}
}

func TestUploadCourseImage_UnsupportedExtension(t *testing.T) {
	_, err := UploadCourseImage(adminCtx(), &UploadCourseImageRequest{
		FileName: "preview.gif",
		FileData: []byte("abc"),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestCreateCourse_Success(t *testing.T) {
	req := makeCreateCourseRequest()

	resp, err := CreateCourse(adminCtx(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Course == nil {
		t.Fatal("expected course, got nil")
	}
	if resp.Course.Title != req.Title {
		t.Errorf("expected title %q, got %q", req.Title, resp.Course.Title)
	}
	if !reflect.DeepEqual(resp.Course.CategoryIDs, req.CategoryIDs) {
		t.Errorf("expected category_ids %v, got %v", req.CategoryIDs, resp.Course.CategoryIDs)
	}
	if resp.Course.Description == nil || *resp.Course.Description != *req.Description {
		t.Errorf("expected description %q, got %v", *req.Description, resp.Course.Description)
	}
	if resp.Course.Lecturer == nil || *resp.Course.Lecturer != *req.Lecturer {
		t.Errorf("expected lecturer %q, got %v", *req.Lecturer, resp.Course.Lecturer)
	}
	if resp.Course.ScormURL != req.ScormURL {
		t.Errorf("expected scorm_url %q, got %q", req.ScormURL, resp.Course.ScormURL)
	}
	if !resp.Course.IsActive {
		t.Error("expected new course to be active")
	}

	row := mustGetCourseRow(t, resp.Course.ID)
	if row.ClientID.String() != testCompanyID {
		t.Errorf("expected client_id %q, got %q", testCompanyID, row.ClientID.String())
	}
	if !reflect.DeepEqual(row.CategoryIds, req.CategoryIDs) {
		t.Errorf("expected stored category_ids %v, got %v", req.CategoryIDs, row.CategoryIds)
	}
}

func TestCreateCourse_WithImageURLSuccess(t *testing.T) {
	req := makeCreateCourseRequest()
	req.ImageURL = strPtr("course-images/preview.png")

	resp, err := CreateCourse(adminCtx(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Course.ImageURL == nil || *resp.Course.ImageURL != *req.ImageURL {
		t.Fatalf("expected image_url %q, got %v", *req.ImageURL, resp.Course.ImageURL)
	}

	row := mustGetCourseRow(t, resp.Course.ID)
	if row.ImageURL == nil || *row.ImageURL != *req.ImageURL {
		t.Fatalf("expected stored image_url %q, got %v", *req.ImageURL, row.ImageURL)
	}
}

func TestCreateCourse_InvalidInput(t *testing.T) {
	_, err := CreateCourse(adminCtx(), &CreateCourseRequest{
		Title:       "Valid Title",
		CategoryIDs: nil,
		ScormURL:    "scorm/uploads/test.zip",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestCreateCourse_EmployeeDenied(t *testing.T) {
	_, err := CreateCourse(employeeCtx(), makeCreateCourseRequest())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.PermissionDenied {
		t.Errorf("expected PermissionDenied, got %v", errs.Code(err))
	}
}

func TestListCourses_AdminSeesActiveAndInactive(t *testing.T) {
	active := mustCreateCourse(t)
	inactive := mustCreateCourse(t)

	if err := DeleteCourse(adminCtx(), inactive.ID.String()); err != nil {
		t.Fatalf("DeleteCourse setup failed: %v", err)
	}

	resp, err := ListCourses(adminCtx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundActive := false
	foundInactive := false
	for _, course := range resp.Courses {
		if course.ID == active.ID {
			foundActive = true
		}
		if course.ID == inactive.ID {
			foundInactive = true
		}
	}

	if !foundActive {
		t.Error("expected active course in admin list")
	}
	if !foundInactive {
		t.Error("expected inactive course in admin list")
	}
}

func TestListCourses_EmployeeSeesOnlyActiveCourses(t *testing.T) {
	active := mustCreateCourse(t)
	inactive := mustCreateCourse(t)

	if err := DeleteCourse(adminCtx(), inactive.ID.String()); err != nil {
		t.Fatalf("DeleteCourse setup failed: %v", err)
	}

	resp, err := ListCourses(employeeCtx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, course := range resp.Courses {
		if !course.IsActive {
			t.Fatalf("employee list should not include inactive course %s", course.ID)
		}
	}

	foundActive := false
	foundInactive := false
	for _, course := range resp.Courses {
		if course.ID == active.ID {
			foundActive = true
		}
		if course.ID == inactive.ID {
			foundInactive = true
		}
	}

	if !foundActive {
		t.Error("expected active course in employee list")
	}
	if foundInactive {
		t.Error("did not expect inactive course in employee list")
	}
}

func TestGetCourse_Success(t *testing.T) {
	created := mustCreateCourse(t)

	resp, err := GetCourse(adminCtx(), created.ID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Course == nil {
		t.Fatal("expected course, got nil")
	}
	if resp.Course.ID != created.ID {
		t.Errorf("expected id %q, got %q", created.ID, resp.Course.ID)
	}
	if !reflect.DeepEqual(resp.Course.CategoryIDs, created.CategoryIDs) {
		t.Errorf("expected category_ids %v, got %v", created.CategoryIDs, resp.Course.CategoryIDs)
	}
}

func TestGetCourse_InvalidID(t *testing.T) {
	_, err := GetCourse(adminCtx(), "not-a-uuid")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestGetCourse_EmployeeCannotGetInactiveCourse(t *testing.T) {
	created := mustCreateCourse(t)

	if err := DeleteCourse(adminCtx(), created.ID.String()); err != nil {
		t.Fatalf("DeleteCourse setup failed: %v", err)
	}

	_, err := GetCourse(employeeCtx(), created.ID.String())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestUpdateCourse_Success(t *testing.T) {
	created := mustCreateCourse(t)

	title := "Updated " + uuid.NewString()
	description := "Updated description"
	lecturer := "Updated lecturer"
	scormURL := "scorm/uploads/" + uuid.NewString() + ".zip"
	categoryIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	resp, err := UpdateCourse(adminCtx(), created.ID.String(), &UpdateCourseRequest{
		Title:       &title,
		CategoryIDs: &categoryIDs,
		Description: &description,
		Lecturer:    &lecturer,
		ScormURL:    &scormURL,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Course.Title != title {
		t.Errorf("expected title %q, got %q", title, resp.Course.Title)
	}
	if !reflect.DeepEqual(resp.Course.CategoryIDs, categoryIDs) {
		t.Errorf("expected category_ids %v, got %v", categoryIDs, resp.Course.CategoryIDs)
	}
	if resp.Course.Description == nil || *resp.Course.Description != description {
		t.Errorf("expected description %q, got %v", description, resp.Course.Description)
	}
	if resp.Course.Lecturer == nil || *resp.Course.Lecturer != lecturer {
		t.Errorf("expected lecturer %q, got %v", lecturer, resp.Course.Lecturer)
	}
	if resp.Course.ScormURL != scormURL {
		t.Errorf("expected scorm_url %q, got %q", scormURL, resp.Course.ScormURL)
	}

	row := mustGetCourseRow(t, created.ID)
	if !reflect.DeepEqual(row.CategoryIds, categoryIDs) {
		t.Errorf("expected stored category_ids %v, got %v", categoryIDs, row.CategoryIds)
	}
}

func TestUpdateCourse_ImageURLSuccess(t *testing.T) {
	created := mustCreateCourse(t)
	imageURL := "course-images/updated.webp"

	resp, err := UpdateCourse(adminCtx(), created.ID.String(), &UpdateCourseRequest{
		ImageURL: &imageURL,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Course.ImageURL == nil || *resp.Course.ImageURL != imageURL {
		t.Fatalf("expected image_url %q, got %v", imageURL, resp.Course.ImageURL)
	}

	row := mustGetCourseRow(t, created.ID)
	if row.ImageURL == nil || *row.ImageURL != imageURL {
		t.Fatalf("expected stored image_url %q, got %v", imageURL, row.ImageURL)
	}
}

func TestUpdateCourse_InvalidCategoryIDs(t *testing.T) {
	created := mustCreateCourse(t)
	categoryIDs := []uuid.UUID{}

	_, err := UpdateCourse(adminCtx(), created.ID.String(), &UpdateCourseRequest{
		CategoryIDs: &categoryIDs,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestUpdateCourse_EmployeeDenied(t *testing.T) {
	created := mustCreateCourse(t)
	title := "Employee Attempt"

	_, err := UpdateCourse(employeeCtx(), created.ID.String(), &UpdateCourseRequest{
		Title: &title,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.PermissionDenied {
		t.Errorf("expected PermissionDenied, got %v", errs.Code(err))
	}
}

func TestDeleteCourse_SoftDeleteSetsInactive(t *testing.T) {
	created := mustCreateCourse(t)

	if err := DeleteCourse(adminCtx(), created.ID.String()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	row := mustGetCourseRow(t, created.ID)
	if row.IsActive {
		t.Error("expected is_active=false after soft delete")
	}
}

func TestDeleteCourse_NotFound(t *testing.T) {
	err := DeleteCourse(adminCtx(), uuid.Nil.String())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestDeleteCourse_EmployeeDenied(t *testing.T) {
	created := mustCreateCourse(t)

	err := DeleteCourse(employeeCtx(), created.ID.String())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.PermissionDenied {
		t.Errorf("expected PermissionDenied, got %v", errs.Code(err))
	}
}

func TestDeleteCourse_EmployeeNoLongerSeesDeletedCourse(t *testing.T) {
	created := mustCreateCourse(t)

	if err := DeleteCourse(adminCtx(), created.ID.String()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	listResp, err := ListCourses(employeeCtx())
	if err != nil {
		t.Fatalf("ListCourses failed: %v", err)
	}
	for _, course := range listResp.Courses {
		if course.ID == created.ID {
			t.Fatalf("employee should not see deleted course %s in list", created.ID)
		}
	}

	_, err = GetCourse(employeeCtx(), created.ID.String())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestListCourses_ReturnsNonNilSlice(t *testing.T) {
	resp, err := ListCourses(adminCtx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Courses == nil {
		t.Error("expected []Course{}, got nil")
	}
}

func TestUploadSCORM_EmptyFileName(t *testing.T) {
	_, err := UploadSCORM(adminCtx(), &UploadSCORMRequest{
		FileName: "   ",
		FileData: []byte("abc"),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestCreateCourse_EmptyTitle(t *testing.T) {
	req := makeCreateCourseRequest()
	req.Title = "   "

	_, err := CreateCourse(adminCtx(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestUpdateCourse_EmptyScormURL(t *testing.T) {
	created := mustCreateCourse(t)
	empty := "   "

	_, err := UpdateCourse(adminCtx(), created.ID.String(), &UpdateCourseRequest{
		ScormURL: &empty,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestUploadSCORM_SanitizesPathTraversalInReturnedObjectKey(t *testing.T) {
	resp, err := UploadSCORM(adminCtx(), &UploadSCORMRequest{
		FileName: "../nested/evil package.zip",
		FileData: []byte("abc"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(resp.ScormURL, "..") {
		t.Fatalf("expected sanitized object key, got %q", resp.ScormURL)
	}
	if !strings.HasPrefix(resp.ScormURL, "scorm/uploads/") {
		t.Fatalf("expected uploads prefix, got %q", resp.ScormURL)
	}
}
