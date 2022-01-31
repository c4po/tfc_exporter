package exporter

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-kit/kit/log/level"
	"github.com/hashicorp/go-tfe"
	_ "net/http/pprof"
)

func (e *Exporter) getAllWorkspaces() ([]*tfe.Workspace, error) {
	level.Debug(e.logger).Log("msg", fmt.Sprintf("Getting all workspaces for orginication %s", e.organization))

	workspaces := make([]*tfe.Workspace, 0)

	count := 0
	totalCount := 0
	page := 0

	for {
		level.Debug(e.logger).Log("msg", fmt.Sprintf("Requesting page %v (items %v %v)", page, count, totalCount))
		response, err := e.client.Workspaces.List(context.Background(), e.organization, tfe.WorkspaceListOptions{
			ListOptions: tfe.ListOptions{
				PageNumber: page,
				PageSize:   100,
			},
			Search:  tfe.String(""),
			Tags:    tfe.String(""),
			Include: tfe.String(""),
		})
		if err != nil {
			return nil, err
		}

		level.Debug(e.logger).Log("msg", fmt.Sprintf("Got %d of %d; next page: %d", response.CurrentPage, response.TotalPages, response.NextPage))

		workspaces = append(workspaces, response.Items...)
		totalCount = response.TotalCount
		page = response.NextPage
		count += len(response.Items)
		if count >= response.TotalCount {
			break
		}
	}
	return workspaces, nil
}

func (e *Exporter) getAllRuns(workspace *tfe.Workspace) ([]*tfe.Run, error) {
	level.Debug(e.logger).Log("msg", fmt.Sprintf("Getting all runs for workspace %s", workspace.Name))
	runs := make([]*tfe.Run, 0)

	count := 0
	totalCount := 0
	page := 0

	for {
		level.Debug(e.logger).Log("msg", fmt.Sprintf("Requesting page %v (items %v %v)", page, count, totalCount))
		response, err := e.client.Runs.List(context.Background(), workspace.ID, tfe.RunListOptions{
			ListOptions: tfe.ListOptions{
				PageNumber: page,
				PageSize:   0, // uses default page size
			},
			Include: tfe.String(""),
		})
		if err != nil {
			return nil, err
		}

		level.Debug(e.logger).Log("msg", fmt.Sprintf("Got %d of %d; next page: %d", response.CurrentPage, response.TotalPages, response.NextPage))

		runs = append(runs, response.Items...)
		totalCount = response.TotalCount
		page = response.NextPage
		count += len(response.Items)
		if count >= response.TotalCount {
			break
		}
	}
	return runs, nil
}

func (e *Exporter) getLastRun(workspace *tfe.Workspace) (*tfe.Run, error) {
	level.Debug(e.logger).Log("msg", fmt.Sprintf("Getting last run for workspace %s", workspace.Name))
	response, err := e.client.Runs.List(context.Background(), workspace.ID, tfe.RunListOptions{
		ListOptions: tfe.ListOptions{
			PageNumber: 0,
			PageSize:   1,
		},
		Include: tfe.String(""),
	})
	if err != nil {
		return nil, err
	}
	if len(response.Items) > 0 {
		return response.Items[0], nil
	}
	return nil, errors.New("No runs found for workspace")
}
