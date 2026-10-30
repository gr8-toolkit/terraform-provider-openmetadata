// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/gr8-toolkit/terraform-provider-openmetadata/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &IngestionPipelineResource{}
var _ resource.ResourceWithImportState = &IngestionPipelineResource{}

// IngestionPipelineCollection is the OM API path segment for ingestion pipelines
// (called "agents" in the OpenMetadata 2.x UI).
const IngestionPipelineCollection = "services/ingestionPipelines"

// IngestionPipelineTypes lists the supported values of the OM `pipelineType` field.
var IngestionPipelineTypes = []string{
	"metadata", "usage", "lineage", "profiler", "autoClassification",
	"TestSuite", "dataInsight", "elasticSearchReindex", "dbt", "application",
}

// IngestionLogLevels lists the supported values of the OM `loggerLevel` field.
var IngestionLogLevels = []string{"DEBUG", "INFO", "WARN", "ERROR"}

// ingestionServiceCollections maps the OM entity type of the entity an ingestion
// pipeline is attached to onto the API collection used to resolve it by name.
var ingestionServiceCollections = map[string]string{
	"databaseService":  databaseServiceCollection,
	"messagingService": messagingServiceCollection,
	"dashboardService": dashboardServiceCollection,
	"pipelineService":  pipelineServiceCollection,
	"mlmodelService":   mlmodelServiceCollection,
	"storageService":   storageServiceCollection,
	"searchService":    searchServiceCollection,
	"metadataService":  metadataServiceCollection,
	"apiService":       apiServiceCollection,
	"driveService":     driveServiceCollection,
	"testSuite":        "dataQuality/testSuites",
}

// IngestionServiceTypes returns the sorted entity types accepted in `service.type`.
func IngestionServiceTypes() []string {
	out := make([]string, 0, len(ingestionServiceCollections))
	for k := range ingestionServiceCollections {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// airflowField describes one attribute of the `airflow_config` object.
type airflowField struct {
	tf   string    // Terraform attribute name
	api  string    // OM JSON key
	kind attr.Type // attribute type
}

var airflowConfigFields = []airflowField{
	{"schedule_interval", "scheduleInterval", types.StringType},
	{"pause_pipeline", "pausePipeline", types.BoolType},
	{"concurrency", "concurrency", types.Int64Type},
	{"retries", "retries", types.Int64Type},
	{"retry_delay", "retryDelay", types.Int64Type},
	{"pipeline_catchup", "pipelineCatchup", types.BoolType},
	{"pipeline_timezone", "pipelineTimezone", types.StringType},
	{"max_active_runs", "maxActiveRuns", types.Int64Type},
	{"workflow_timeout", "workflowTimeout", types.Int64Type},
	{"email", "email", types.StringType},
}

// IngestionServiceAttrTypes is the object type of the `service` attribute.
func IngestionServiceAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{"type": types.StringType, "name": types.StringType}
}

// AirflowConfigAttrTypes is the object type of the `airflow_config` attribute.
func AirflowConfigAttrTypes() map[string]attr.Type {
	m := make(map[string]attr.Type, len(airflowConfigFields))
	for _, f := range airflowConfigFields {
		m[f.tf] = f.kind
	}
	return m
}

// IngestionServiceObject converts the `service` entity reference of an API
// response into the `service` object. It returns a null object when absent.
func IngestionServiceObject(data map[string]interface{}) types.Object {
	ref, ok := data["service"].(map[string]interface{})
	if !ok {
		return types.ObjectNull(IngestionServiceAttrTypes())
	}
	name := StringVal(ref, "fullyQualifiedName")
	if name.IsNull() {
		name = StringVal(ref, "name")
	}
	obj, diags := types.ObjectValue(IngestionServiceAttrTypes(), map[string]attr.Value{
		"type": StringVal(ref, "type"),
		"name": name,
	})
	if diags.HasError() {
		return types.ObjectNull(IngestionServiceAttrTypes())
	}
	return obj
}

// AirflowConfigObject converts the `airflowConfig` of an API response into the
// `airflow_config` object. Fields missing from the response become null.
func AirflowConfigObject(data map[string]interface{}) types.Object {
	cfg, _ := data["airflowConfig"].(map[string]interface{})
	vals := make(map[string]attr.Value, len(airflowConfigFields))
	for _, f := range airflowConfigFields {
		switch f.kind {
		case types.BoolType:
			vals[f.tf] = BoolVal(cfg, f.api)
		case types.Int64Type:
			vals[f.tf] = Int64Val(cfg, f.api)
		default:
			vals[f.tf] = StringVal(cfg, f.api)
		}
	}
	obj, diags := types.ObjectValue(AirflowConfigAttrTypes(), vals)
	if diags.HasError() {
		return types.ObjectNull(AirflowConfigAttrTypes())
	}
	return obj
}

type IngestionPipelineResource struct{ client *client.Client }

type IngestionPipelineResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	DisplayName          types.String `tfsdk:"display_name"`
	Description          types.String `tfsdk:"description"`
	FQN                  types.String `tfsdk:"fully_qualified_name"`
	PipelineType         types.String `tfsdk:"pipeline_type"`
	Service              types.Object `tfsdk:"service"`
	SourceConfigJSON     types.String `tfsdk:"source_config_json"`
	AirflowConfig        types.Object `tfsdk:"airflow_config"`
	LoggerLevel          types.String `tfsdk:"logger_level"`
	RaiseOnError         types.Bool   `tfsdk:"raise_on_error"`
	EnableStreamableLogs types.Bool   `tfsdk:"enable_streamable_logs"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	Deployed             types.Bool   `tfsdk:"deployed"`
	Domains              types.List   `tfsdk:"domains"`
	Owners               types.List   `tfsdk:"owners"`
}

func NewIngestionPipelineResource() resource.Resource { return &IngestionPipelineResource{} }

func (r *IngestionPipelineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ingestion_pipeline"
}

func (r *IngestionPipelineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an OpenMetadata Ingestion Pipeline (an \"agent\" in the OpenMetadata 2.x UI). " +
			"One resource covers every workflow type: metadata, usage, lineage, profiler, auto classification, " +
			"data quality (TestSuite), dbt, and so on. The resource only manages the pipeline definition; " +
			"it does not deploy it to the ingestion runner.",
		Attributes: map[string]schema.Attribute{
			"id":                   IDAttribute(),
			"name":                 NameAttribute(),
			"display_name":         DisplayNameAttribute(),
			"description":          DescriptionAttribute(false),
			"fully_qualified_name": FullyQualifiedNameAttribute(),
			"pipeline_type": schema.StringAttribute{
				Description: "Workflow type of the pipeline. One of: " + joinQuoted(IngestionPipelineTypes) + ".",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{stringvalidator.OneOf(IngestionPipelineTypes...)},
			},
			"service": schema.SingleNestedAttribute{
				Description: "The entity the pipeline ingests from, usually a service. " +
					"The provider resolves it by name to the UUID the OpenMetadata API expects.",
				Required: true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "Entity type of the service. One of: " + joinQuoted(IngestionServiceTypes()) + ".",
						Required:    true,
						Validators:  []validator.String{stringvalidator.OneOf(IngestionServiceTypes()...)},
					},
					"name": schema.StringAttribute{
						Description: "Fully qualified name of the service (for services this is the service name).",
						Required:    true,
					},
				},
			},
			"source_config_json": schema.StringAttribute{
				Description: "The pipeline `sourceConfig` as a JSON string, e.g. " +
					"`{\"config\":{\"type\":\"DatabaseMetadata\"}}`. The structure depends on `pipeline_type`. " +
					"OpenMetadata fills in defaults when it stores the config, so this value is not read back from the API " +
					"and drift in it is not detected.",
				// No NormalizeJSONString here: the value is never read back, so there is
				// no drift to normalise, and Terraform rejects a plan modifier that
				// rewrites the configured value of a non-computed attribute.
				Required: true,
			},
			"airflow_config": schema.SingleNestedAttribute{
				Description: "Scheduling and runtime settings of the pipeline. Unset fields take the OpenMetadata defaults.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"schedule_interval": schema.StringAttribute{
						Description: "Schedule in cron format, e.g. `0 * * * *`. Leave unset for a pipeline that is only run manually.",
						Optional:    true,
					},
					"pause_pipeline": boolAirflowAttr("Pause the pipeline from running once deployed."),
					"concurrency":    int64AirflowAttr("Concurrency of the pipeline."),
					"retries":        int64AirflowAttr("Number of retries in case of failure."),
					"retry_delay":    int64AirflowAttr("Delay between retries, in seconds."),
					"pipeline_catchup": boolAirflowAttr(
						"Run past executions if the start date is in the past."),
					"pipeline_timezone": schema.StringAttribute{
						Description:   "Timezone the schedule is evaluated in.",
						Optional:      true,
						Computed:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"max_active_runs":  int64AirflowAttr("Maximum number of active runs."),
					"workflow_timeout": int64AirflowAttr("Timeout of the workflow, in seconds."),
					"email": schema.StringAttribute{
						Description:   "Email address to notify about the workflow status.",
						Optional:      true,
						Computed:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
				},
			},
			"logger_level": schema.StringAttribute{
				Description:   "Logging level of the workflow. One of: " + joinQuoted(IngestionLogLevels) + ".",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators:    []validator.String{stringvalidator.OneOf(IngestionLogLevels...)},
			},
			"raise_on_error": schema.BoolAttribute{
				Description:   "Mark the workflow as failed if processing errors are encountered.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"enable_streamable_logs": schema.BoolAttribute{
				Description:   "Stream the ingestion logs to the OpenMetadata server while the pipeline runs.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"enabled": schema.BoolAttribute{
				Description: "True if the pipeline runs on its next schedule, false if it is paused.",
				Computed:    true,
			},
			"deployed": schema.BoolAttribute{
				Description: "True once the pipeline has been deployed to the ingestion runner. Read-only.",
				Computed:    true,
			},
			"owners":  OwnersAttribute(),
			"domains": DomainsAttribute(),
		},
	}
}

func boolAirflowAttr(desc string) schema.BoolAttribute {
	return schema.BoolAttribute{
		Description:   desc,
		Optional:      true,
		Computed:      true,
		PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
	}
}

func int64AirflowAttr(desc string) schema.Int64Attribute {
	return schema.Int64Attribute{
		Description:   desc,
		Optional:      true,
		Computed:      true,
		PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
	}
}

func joinQuoted(vals []string) string {
	out := ""
	for i, v := range vals {
		if i > 0 {
			out += ", "
		}
		out += "`" + v + "`"
	}
	return out
}

func (r *IngestionPipelineResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *IngestionPipelineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan IngestionPipelineResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, err := r.buildBody(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building request body", err.Error())
		return
	}
	raw, err := r.client.CreateOrUpdate(ctx, IngestionPipelineCollection, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating ingestion pipeline", err.Error())
		return
	}
	r.readIntoState(raw, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IngestionPipelineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state IngestionPipelineResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	raw, err := r.client.GetByName(ctx, IngestionPipelineCollection, r.lookupFQN(&state), []string{"owners", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error reading ingestion pipeline", err.Error())
		return
	}
	if raw == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	r.readIntoState(raw, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *IngestionPipelineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan IngestionPipelineResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, err := r.buildBody(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building request body", err.Error())
		return
	}
	raw, err := r.client.CreateOrUpdate(ctx, IngestionPipelineCollection, body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating ingestion pipeline", err.Error())
		return
	}
	r.readIntoState(raw, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IngestionPipelineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state IngestionPipelineResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Deleting ingestion pipeline", map[string]interface{}{"fqn": state.FQN.ValueString()})
	if err := r.client.Delete(ctx, IngestionPipelineCollection, state.ID.ValueString(), true); err != nil {
		resp.Diagnostics.AddError("Error deleting ingestion pipeline", err.Error())
	}
}

// ImportState imports by fully qualified name, i.e. "<service>.<pipeline name>".
func (r *IngestionPipelineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	raw, err := r.client.GetByName(ctx, IngestionPipelineCollection, req.ID, []string{"owners", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error importing ingestion pipeline", err.Error())
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Ingestion pipeline not found", fmt.Sprintf("No ingestion pipeline with FQN %q", req.ID))
		return
	}
	state := IngestionPipelineResourceModel{
		Service:       types.ObjectNull(IngestionServiceAttrTypes()),
		AirflowConfig: types.ObjectNull(AirflowConfigAttrTypes()),
		Owners:        OwnersListNull(),
		Domains:       types.ListNull(types.StringType),
	}
	r.readIntoState(raw, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// lookupFQN returns the FQN used to fetch the pipeline, deriving it from the
// service and pipeline name when the state does not hold one yet.
func (r *IngestionPipelineResource) lookupFQN(state *IngestionPipelineResourceModel) string {
	if fqn := state.FQN.ValueString(); fqn != "" {
		return fqn
	}
	if svcName, _ := objectString(state.Service, "name"); svcName != "" {
		return svcName + "." + state.Name.ValueString()
	}
	return state.Name.ValueString()
}

// objectString reads a known string attribute from an object value.
func objectString(obj types.Object, key string) (string, bool) {
	if obj.IsNull() || obj.IsUnknown() {
		return "", false
	}
	s, ok := obj.Attributes()[key].(types.String)
	if !ok || s.IsNull() || s.IsUnknown() {
		return "", false
	}
	return s.ValueString(), true
}

// resolveService turns the `service` attribute into the {id, type} entity
// reference expected by the OM API.
func (r *IngestionPipelineResource) resolveService(ctx context.Context, svc types.Object) (map[string]interface{}, error) {
	svcType, okType := objectString(svc, "type")
	svcName, okName := objectString(svc, "name")
	if !okType || !okName {
		return nil, fmt.Errorf("service.type and service.name must be set")
	}
	collection, ok := ingestionServiceCollections[svcType]
	if !ok {
		return nil, fmt.Errorf("unsupported service type %q", svcType)
	}
	raw, err := r.client.GetByName(ctx, collection, svcName, nil)
	if err != nil {
		return nil, fmt.Errorf("looking up %s %q: %w", svcType, svcName, err)
	}
	if raw == nil {
		return nil, fmt.Errorf("%s %q not found", svcType, svcName)
	}
	data, err := Unmarshal(raw)
	if err != nil {
		return nil, fmt.Errorf("parsing %s %q: %w", svcType, svcName, err)
	}
	id, _ := data["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("%s %q has no id in the API response", svcType, svcName)
	}
	return map[string]interface{}{"id": id, "type": svcType}, nil
}

func (r *IngestionPipelineResource) buildBody(ctx context.Context, plan *IngestionPipelineResourceModel) (map[string]interface{}, error) {
	// Validate local input before making any API call.
	var sourceConfig map[string]interface{}
	if err := json.Unmarshal([]byte(plan.SourceConfigJSON.ValueString()), &sourceConfig); err != nil || sourceConfig == nil {
		return nil, fmt.Errorf("invalid source_config_json: must be a JSON object")
	}
	service, err := r.resolveService(ctx, plan.Service)
	if err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"pipelineType":  plan.PipelineType.ValueString(),
		"service":       service,
		"sourceConfig":  sourceConfig,
		"airflowConfig": buildAirflowConfig(plan.AirflowConfig),
	}
	if !plan.DisplayName.IsNull() && !plan.DisplayName.IsUnknown() {
		body["displayName"] = plan.DisplayName.ValueString()
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		body["description"] = plan.Description.ValueString()
	}
	if !plan.LoggerLevel.IsNull() && !plan.LoggerLevel.IsUnknown() {
		body["loggerLevel"] = plan.LoggerLevel.ValueString()
	}
	if !plan.RaiseOnError.IsNull() && !plan.RaiseOnError.IsUnknown() {
		body["raiseOnError"] = plan.RaiseOnError.ValueBool()
	}
	if !plan.EnableStreamableLogs.IsNull() && !plan.EnableStreamableLogs.IsUnknown() {
		body["enableStreamableLogs"] = plan.EnableStreamableLogs.ValueBool()
	}
	if !plan.Owners.IsNull() && !plan.Owners.IsUnknown() {
		body["owners"] = extractOwnerRefs(ctx, plan.Owners)
	}
	if !plan.Domains.IsNull() && !plan.Domains.IsUnknown() {
		var vals []string
		plan.Domains.ElementsAs(ctx, &vals, false)
		body["domains"] = vals
	}
	return body, nil
}

// buildAirflowConfig converts the `airflow_config` object into the OM
// `airflowConfig` JSON. Null and unknown fields are omitted so the server
// applies its defaults. The API requires the key, so an empty map is returned
// when nothing is set.
func buildAirflowConfig(obj types.Object) map[string]interface{} {
	cfg := map[string]interface{}{}
	if obj.IsNull() || obj.IsUnknown() {
		return cfg
	}
	attrs := obj.Attributes()
	for _, f := range airflowConfigFields {
		switch v := attrs[f.tf].(type) {
		case types.String:
			if !v.IsNull() && !v.IsUnknown() {
				cfg[f.api] = v.ValueString()
			}
		case types.Bool:
			if !v.IsNull() && !v.IsUnknown() {
				cfg[f.api] = v.ValueBool()
			}
		case types.Int64:
			if !v.IsNull() && !v.IsUnknown() {
				cfg[f.api] = v.ValueInt64()
			}
		}
	}
	return cfg
}

func (r *IngestionPipelineResource) readIntoState(raw []byte, state *IngestionPipelineResourceModel) {
	data, err := Unmarshal(raw)
	if err != nil {
		return
	}
	state.ID = StringVal(data, "id")
	state.Name = StringVal(data, "name")
	state.DisplayName = StringVal(data, "displayName")
	state.Description = DescriptionVal(data, state.Description)
	state.FQN = StringVal(data, "fullyQualifiedName")
	state.PipelineType = StringVal(data, "pipelineType")
	state.LoggerLevel = StringVal(data, "loggerLevel")
	state.RaiseOnError = BoolVal(data, "raiseOnError")
	state.EnableStreamableLogs = BoolVal(data, "enableStreamableLogs")
	state.Enabled = BoolVal(data, "enabled")
	state.Deployed = BoolVal(data, "deployed")
	state.AirflowConfig = AirflowConfigObject(data)
	state.Domains = StringListVal(data, "domains")

	// owners is not read back: the plan/state value is preserved. Overwriting it
	// with null would make Terraform fail with "inconsistent result after apply"
	// whenever the user configures owners. ImportState seeds it as null.

	// Keep the planned service when the API omits it so state stays consistent.
	if svc := IngestionServiceObject(data); !svc.IsNull() {
		state.Service = svc
	}

	// source_config_json is deliberately not read back: OM stores the config
	// with server-side defaults filled in, so the response never matches the
	// user's input. The plan/state value is preserved.
}
