package AzureDevopsClient

import (
	"encoding/json"
)

type WorkItem struct {
	Id     int64          `json:"id"`
	Fields WorkItemFields `json:"fields"`
}

type AssignedTo struct {
	DisplayName string `json:"displayName"`
}

type WorkItemFields struct {
	Title         string     `json:"System.Title"`
	AreaPath      string     `json:"System.AreaPath"`
	CreatedDate   string     `json:"System.CreatedDate"`
	AcceptedDate  string     `json:"Microsoft.VSTS.CodeReview.AcceptedDate"`
	ResolvedDate  string     `json:"Microsoft.VSTS.Common.ResolvedDate"`
	ClosedDate    string     `json:"Microsoft.VSTS.Common.ClosedDate"`
	StoryPoints   float64    `json:"Microsoft.VSTS.Scheduling.StoryPoints"`
	WorkItemType  string     `json:"System.WorkItemType"`
	AssignedTo    AssignedTo `json:"System.AssignedTo"`
	Priority      int        `json:"Microsoft.VSTS.Common.Priority"`
	IterationPath string     `json:"System.IterationPath"`
	State         string     `json:"System.State"`
}

func (c *AzureDevopsClient) GetWorkItem(workItemUrl string) (workItem WorkItem, error error) {
	defer c.concurrencyUnlock()
	c.concurrencyLock()

	response, err := c.rest().R().Get(workItemUrl)
	if err := c.checkResponse(response, err); err != nil {
		error = err
		return
	}

	err = json.Unmarshal(response.Body(), &workItem)
	if err != nil {
		error = err
	}

	return
}
