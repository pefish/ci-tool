package constant

import go_best_type "github.com/pefish/go-best-type"

const (
	PARAM_ERROR uint64 = 1001 // 参数错误
)

const (
	ActionType_CI      go_best_type.ActionType = "ci"
	ActionType_LOG     go_best_type.ActionType = "log"
	ActionType_ReadLog go_best_type.ActionType = "read_log"
)
