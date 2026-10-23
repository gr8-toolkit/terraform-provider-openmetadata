// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package datasources

import (
	"context"
	"fmt"

	"github.com/gr8-toolkit/terraform-provider-openmetadata/internal/client"
	"github.com/gr8-toolkit/terraform-provider-openmetadata/internal/resources"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &IngestionPipelineDataSource{}

type IngestionPipelineDataSource struct{ client *client.Client }

// IngestionPipelineDataSourceModel uses name as the full FQN ("<service>.<pipeline>"),
// because pipeline names are only unique within a service.
type IngestionPipelineDataSourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	DisplayName          types.String `tfsdk:"display_name"`
	Description          types.String `tfsdk:"description"`
	FQN                  types.String `tfsdk:"fully_qualified_name"`
	PipelineType         types.String `tfsdk:"pipeline_type"`
	Service              types.Object `tfsdk:"service"`
	AirflowConfig        types.Object `tfsdk:"airflow_config"`
	LoggerLevel          types.String `tfsdk:"logger_level"`
	RaiseOnError         types.Bool   `tfsdk:"raise_on_error"`
	EnableStreamableLogs types.Bool   `tfsdk:"enable_streamable_logs"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	Deployed             types.Bool   `tfsdk:"deployed"`
	Domains              types.List   `tfsdk:"domains"`
	Owners               types.List   `tfsdk:"owners"`
}

func NewIngestionPipelineDataSource() datasource.DataSource { return &IngestionPipelineDataSource{} }

func (d *IngestionPipelineDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ingestion_pipeline"
}

func (d *IngestionPipelineDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches an OpenMetadata Ingestion Pipeline (an \"agent\" in the OpenMetadata 2.x UI) by its " +
			"fully qualified name (service.pipeline). The `sourceConfig` is not returned.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, Description: "UUID of the ingestion pipeline."},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Fully qualified name of the pipeline, e.g. \"my_service.my_pipeline\".",
			},
			"display_name":         schema.StringAttribute{Computed: true, Description: "Human-readable display name."},
			"description":          schema.StringAttribute{Computed: true, Description: "Description of the pipeline."},
			"fully_qualified_name": schema.StringAttribute{Computed: true, Description: "Fully qualified name."},
			"pipeline_type":        schema.StringAttribute{Computed: true, Description: "Workflow type (metadata, usage, lineage, profiler, ...)."},
			"service": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "The entity the pipeline ingests from.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{Computed: true, Description: "Entity type of the service."},
					"name": schema.StringAttribute{Computed: true, Description: "Fully qualified name of the service."},
				},
			},
			"airflow_config": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Scheduling and runtime settings of the pipeline.",
				Attributes: map[string]schema.Attribute{
					"schedule_interval": schema.StringAttribute{Computed: true, Description: "Schedule in cron format."},
					"pause_pipeline":    schema.BoolAttribute{Computed: true, Description: "Whether the pipeline is paused once deployed."},
					"concurrency":       schema.Int64Attribute{Computed: true, Description: "Concurrency of the pipeline."},
					"retries":           schema.Int64Attribute{Computed: true, Description: "Number of retries in case of failure."},
					"retry_delay":       schema.Int64Attribute{Computed: true, Description: "Delay between retries, in seconds."},
					"pipeline_catchup":  schema.BoolAttribute{Computed: true, Description: "Whether past executions are run."},
					"pipeline_timezone": schema.StringAttribute{Computed: true, Description: "Timezone of the schedule."},
					"max_active_runs":   schema.Int64Attribute{Computed: true, Description: "Maximum number of active runs."},
					"workflow_timeout":  schema.Int64Attribute{Computed: true, Description: "Workflow timeout, in seconds."},
					"email":             schema.StringAttribute{Computed: true, Description: "Email notified about the workflow status."},
				},
			},
			"logger_level":           schema.StringAttribute{Computed: true, Description: "Logging level of the workflow."},
			"raise_on_error":         schema.BoolAttribute{Computed: true, Description: "Whether processing errors fail the workflow."},
			"enable_streamable_logs": schema.BoolAttribute{Computed: true, Description: "Whether logs are streamed to the server."},
			"enabled":                schema.BoolAttribute{Computed: true, Description: "False if the pipeline is paused."},
			"deployed":               schema.BoolAttribute{Computed: true, Description: "True once the pipeline has been deployed."},
			"domains": schema.ListAttribute{
				Computed: true, Description: "Fully qualified names of the domains this pipeline belongs to.",
				ElementType: types.StringType,
			},
			"owners": schema.ListNestedAttribute{
				Computed: true, Description: "Owners of this pipeline.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true, Description: "UUID of the owner entity."},
						"type": schema.StringAttribute{Computed: true, Description: "Type of the owner (user or team)."},
					},
				},
			},
		},
	}
}

func (d *IngestionPipelineDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *IngestionPipelineDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state IngestionPipelineDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	raw, err := d.client.GetByName(ctx, resources.IngestionPipelineCollection, state.Name.ValueString(), []string{"owners", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error reading ingestion pipeline", err.Error())
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Ingestion pipeline not found",
			fmt.Sprintf("no ingestion pipeline with FQN %q found", state.Name.ValueString()))
		return
	}
	data, err := resources.Unmarshal(raw)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing ingestion pipeline response", err.Error())
		return
	}
	state.ID = resources.StringVal(data, "id")
	state.DisplayName = resources.StringVal(data, "displayName")
	state.Description = resources.StringVal(data, "description")
	state.FQN = resources.StringVal(data, "fullyQualifiedName")
	state.PipelineType = resources.StringVal(data, "pipelineType")
	state.Service = resources.IngestionServiceObject(data)
	state.AirflowConfig = resources.AirflowConfigObject(data)
	state.LoggerLevel = resources.StringVal(data, "loggerLevel")
	state.RaiseOnError = resources.BoolVal(data, "raiseOnError")
	state.EnableStreamableLogs = resources.BoolVal(data, "enableStreamableLogs")
	state.Enabled = resources.BoolVal(data, "enabled")
	state.Deployed = resources.BoolVal(data, "deployed")
	state.Domains = resources.StringListVal(data, "domains")
	state.Owners = resources.OwnersListFromRefs(resources.ParseEntityRefs(data, "owners"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
