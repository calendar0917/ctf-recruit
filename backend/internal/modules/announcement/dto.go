package announcement

type CreateAnnouncementRequest struct {
	Title       string  `json:"title"`
	Content     string  `json:"content"`
	IsPublished bool    `json:"isPublished"`
	PublishedAt *string `json:"publishedAt"`
}

type UpdateAnnouncementRequest struct {
	Title       *string `json:"title"`
	Content     *string `json:"content"`
	IsPublished *bool   `json:"isPublished"`
	PublishedAt *string `json:"publishedAt"`
}

type AnnouncementResponse struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Content     string  `json:"content"`
	IsPublished bool    `json:"isPublished"`
	PublishedAt *string `json:"publishedAt,omitempty"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

type AnnouncementListResponse struct {
	Items  []AnnouncementResponse `json:"items"`
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
}
