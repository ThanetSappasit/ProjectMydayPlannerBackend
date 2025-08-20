package dto

type CreateBoardRequest struct {
	BoardName string `json:"board_name" validate:"required"`
	Is_group  string `json:"is_group"`
}
