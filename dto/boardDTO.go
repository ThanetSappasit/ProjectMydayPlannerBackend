package dto

type CreateBoardRequest struct {
	BoardName string `json:"boardname" validate:"required"`
	Is_group  string `json:"isgroup"`
}
