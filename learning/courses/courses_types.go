package courses

import "time"

type CourseFormat string

const (
	CourseFormatOnline    CourseFormat = "online"
	CourseFormatOffline   CourseFormat = "offline"
	CourseFormatHybrid    CourseFormat = "hybrid"
	CourseFormatWebinar   CourseFormat = "webinar"
	CourseFormatSelfPaced CourseFormat = "self_paced"
)

func (f CourseFormat) IsValid() bool {
	switch f {
	case "", CourseFormatOnline, CourseFormatOffline, CourseFormatHybrid, CourseFormatWebinar, CourseFormatSelfPaced:
		return true
	}
	return false
}

type Course struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Description *string       `json:"description,omitempty"`
	Format      *CourseFormat `json:"format,omitempty"`
	Category    *string       `json:"category,omitempty"`
	IsExternal  bool          `json:"is_external"`
	IsActive    bool          `json:"is_active"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

type CourseModule struct {
	ID              string    `json:"id"`
	CourseID        string    `json:"course_id"`
	Title           string    `json:"title"`
	Description     *string   `json:"description,omitempty"`
	SortOrder       int       `json:"sort_order"`
	DurationMinutes *int      `json:"duration_minutes,omitempty"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CreateCourseRequest struct {
	Title       string        `json:"title"`
	Description *string       `json:"description,omitempty"`
	Format      *CourseFormat `json:"format,omitempty"`
	Category    *string       `json:"category,omitempty"`
	IsExternal  *bool         `json:"is_external,omitempty"`
}

type UpdateCourseRequest struct {
	Title       *string       `json:"title,omitempty"`
	Description *string       `json:"description,omitempty"`
	Format      *CourseFormat `json:"format,omitempty"`
	Category    *string       `json:"category,omitempty"`
	IsExternal  *bool         `json:"is_external,omitempty"`
	IsActive    *bool         `json:"is_active,omitempty"`
}

type CreateCourseModuleRequest struct {
	CourseID        string  `json:"course_id"`
	Title           string  `json:"title"`
	Description     *string `json:"description,omitempty"`
	SortOrder       *int    `json:"sort_order,omitempty"`
	DurationMinutes *int    `json:"duration_minutes,omitempty"`
}

type UpdateCourseModuleRequest struct {
	Title           *string `json:"title,omitempty"`
	Description     *string `json:"description,omitempty"`
	SortOrder       *int    `json:"sort_order,omitempty"`
	DurationMinutes *int    `json:"duration_minutes,omitempty"`
	IsActive        *bool   `json:"is_active,omitempty"`
}

type GetCourseResponse struct {
	Course  Course         `json:"course"`
	Modules []CourseModule `json:"modules"`
}

type ListCoursesResponse struct {
	Courses []Course `json:"courses"`
	Total   int      `json:"total"`
}

type GetCourseModuleResponse struct {
	Module CourseModule `json:"module"`
}

type ListCourseModulesResponse struct {
	Modules []CourseModule `json:"modules"`
	Total   int            `json:"total"`
}

type DeleteCourseResponse struct {
	Message string `json:"message"`
}

type DeleteCourseModuleResponse struct {
	Message string `json:"message"`
}
