package agent_tool_svc

type ToolItem struct {
	ID           int64  `json:"id"`
	AgentID      int64  `json:"agentId"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	ParamSchema  string `json:"paramSchema"`
	ExecutorType string `json:"executorType"`
	ExecutorConf string `json:"executorConf"`
	Enabled      bool   `json:"enabled"`
	SortOrder    int    `json:"sortOrder"`
	Createtime   int64  `json:"createtime"`
	Updatetime   int64  `json:"updatetime"`
}

type CreateToolRequest struct {
	AgentID      int64  `json:"agentId"`
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description"`
	ParamSchema  string `json:"paramSchema"`
	ExecutorType string `json:"executorType" binding:"required"`
	ExecutorConf string `json:"executorConf"`
}

type CreateToolResponse struct {
	Item *ToolItem `json:"item"`
}

type UpdateToolRequest struct {
	ID           int64  `json:"id" binding:"required"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	ParamSchema  string `json:"paramSchema"`
	ExecutorType string `json:"executorType"`
	ExecutorConf string `json:"executorConf"`
	Enabled      *bool  `json:"enabled"`
}

type UpdateToolResponse struct {
	Item *ToolItem `json:"item"`
}

type DeleteToolRequest struct {
	ID int64 `json:"id" binding:"required"`
}

type DeleteToolResponse struct{}

type ListToolsRequest struct {
	AgentID int64 `json:"agentId"`
}

type ListToolsResponse struct {
	Items []*ToolItem `json:"items"`
}
