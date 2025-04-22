package dto

type CreateBoardRequest struct {
	BoardName string `json:"board_name" validate:"required"`
	CreatedBy string `json:"created_by" validate:"required"`
	Is_group  string `json:"is_group"`
}

type GetBoardsRequest struct {
	Email string `json:"UserId" validate:"required"`
	Group string `json:"is_group"`
}
