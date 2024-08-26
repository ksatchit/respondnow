package incident

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/respondnow/respond/server/pkg/auth"

	"github.com/google/uuid"
	"github.com/respondnow/respond/server/pkg/database/mongodb"
	incidentdb "github.com/respondnow/respond/server/pkg/database/mongodb/incident"
	"github.com/respondnow/respond/server/pkg/incident"
	"github.com/respondnow/respond/server/utils"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

const (
	// How many total characters are allowed in a Slack channel name?
	// Limit the channel name to 76 to take this into account
	slackChannelNameLengthCap = 80
	// How many characters does the incident prefix take up?
	channelNamePrefixLength = len("rn-20240820150405-")
	// How long can the provided description be?
	incidentDescriptionMaxLength = slackChannelNameLengthCap - channelNamePrefixLength
)

func getSeverityBlock() *slack.InputBlock {
	supportedIncidentSeverities := incidentdb.NewIncidentOperator(mongodb.Operator).GetIncidentSeverities()
	initialOptionForIncidentSeverity := string(supportedIncidentSeverities[len(supportedIncidentSeverities)-1])
	incidentSevOptions := []*slack.OptionBlockObject{}
	for _, incidentSev := range supportedIncidentSeverities {
		incidentSevOptions = append(incidentSevOptions, slack.NewOptionBlockObject(
			string(incidentSev),
			slack.NewTextBlockObject(slack.PlainTextType,
				string(incidentSev), true, false),
			slack.NewTextBlockObject(slack.PlainTextType,
				string(incidentSev), true, false),
		))
	}

	return slack.NewInputBlock(
		"incident_severity",
		&slack.TextBlockObject{
			Type:  slack.PlainTextType,
			Text:  ":vertical_traffic_light: Severity",
			Emoji: false,
		},
		nil,
		&slack.SelectBlockElement{
			Type:     slack.OptTypeStatic,
			ActionID: "create_incident_modal_set_incident_severity",
			Placeholder: slack.NewTextBlockObject(slack.PlainTextType,
				"Select severity of the incident...", false, false),
			InitialOption: slack.NewOptionBlockObject(
				initialOptionForIncidentSeverity,
				slack.NewTextBlockObject(slack.PlainTextType,
					initialOptionForIncidentSeverity, false, false),
				slack.NewTextBlockObject(slack.PlainTextType,
					initialOptionForIncidentSeverity, false, false),
			),
			Options: incidentSevOptions,
		},
	)
}

func getNameBlock() *slack.InputBlock {
	return slack.NewInputBlock("create_incident_modal_name", slack.NewTextBlockObject(
		slack.PlainTextType, ":writing_hand: Incident Name", false, false,
	), nil, slack.PlainTextInputBlockElement{
		Type:      slack.METPlainTextInput,
		MaxLength: incidentDescriptionMaxLength,
		ActionID:  "create_incident_modal_set_name",
		Placeholder: slack.NewTextBlockObject(slack.PlainTextType, "IAM service is down",
			false, false),
	})
}

func getSummaryBlock() *slack.InputBlock {
	return slack.NewInputBlock("create_incident_modal_summary", slack.NewTextBlockObject(
		slack.PlainTextType, ":memo: Summary", false, false,
	), nil, slack.PlainTextInputBlockElement{
		Type:      slack.METPlainTextInput,
		Multiline: true,
		ActionID:  "create_incident_modal_set_summary",
		Placeholder: slack.NewTextBlockObject(slack.PlainTextType, "A brief description of the problem.",
			false, false),
	})
}

func getChannelSelectBlock() *slack.InputBlock {
	return slack.NewInputBlock("create_incident_modal_conversation_select", slack.NewTextBlockObject(
		slack.PlainTextType, "Select a channel to post the result on", false, false,
	), nil, slack.SelectBlockElement{
		Type:               slack.OptTypeChannels,
		ActionID:           "create_incident_modal_select_conversation",
		ResponseURLEnabled: true,
	})
}

func getRoleBlock() *slack.InputBlock {
	supportedRoles := incidentdb.NewIncidentOperator(mongodb.Operator).GetIncidentRoles()
	initialOptionForIncidentRole := string(supportedRoles[0])
	incidentRoleOptions := []*slack.OptionBlockObject{}
	for _, role := range supportedRoles {
		incidentRoleOptions = append(incidentRoleOptions, slack.NewOptionBlockObject(
			string(role),
			slack.NewTextBlockObject(slack.PlainTextType,
				string(role), false, false),
			slack.NewTextBlockObject(slack.PlainTextType,
				string(role), false, false),
		))
	}

	return slack.NewInputBlock(
		"incident_role",
		&slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: ":firefighter: Assign role to yourself",
		},
		nil,
		&slack.MultiSelectBlockElement{
			Type:     slack.MultiOptTypeStatic,
			ActionID: "create_incident_modal_set_incident_role",
			Placeholder: slack.NewTextBlockObject(slack.PlainTextType, "Assign role to yourself...",
				false, false),
			InitialOptions: []*slack.OptionBlockObject{
				slack.NewOptionBlockObject(
					initialOptionForIncidentRole,
					slack.NewTextBlockObject(slack.PlainTextType,
						initialOptionForIncidentRole, false, false),
					slack.NewTextBlockObject(slack.PlainTextType,
						initialOptionForIncidentRole, false, false),
				),
			},
			Options: incidentRoleOptions,
		},
	)
}

func getTypeBlock() *slack.InputBlock {
	supportedIncidentTypes := incidentdb.NewIncidentOperator(mongodb.Operator).GetIncidentTypes()
	initialOptionForIncidentType := string(supportedIncidentTypes[0])
	incidentTypeOptions := []*slack.OptionBlockObject{}
	for _, incidentType := range supportedIncidentTypes {
		incidentTypeOptions = append(incidentTypeOptions, slack.NewOptionBlockObject(
			string(incidentType),
			slack.NewTextBlockObject(slack.PlainTextType,
				string(incidentType), false, false),
			slack.NewTextBlockObject(slack.PlainTextType,
				string(incidentType), false, false),
		))
	}

	return slack.NewInputBlock(
		"incident_type",
		&slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: ":fire: Incident Type",
		},
		nil,
		&slack.SelectBlockElement{
			Type:        slack.OptTypeStatic,
			ActionID:    "create_incident_modal_set_incident_type",
			Placeholder: slack.NewTextBlockObject(slack.PlainTextType, "Select incident type...", false, false),
			InitialOption: slack.NewOptionBlockObject(
				initialOptionForIncidentType,
				slack.NewTextBlockObject(slack.PlainTextType,
					initialOptionForIncidentType, false, false),
				slack.NewTextBlockObject(slack.PlainTextType,
					initialOptionForIncidentType, false, false),
			),
			Options: incidentTypeOptions,
		},
	)
}

// func getStageBlock() *slack.SectionBlock {
// 	supportedIncidentTypes := incidentdb.NewIncidentOperator(mongodb.Operator).GetIncidentTypes()
// 	initialOptionForIncidentType := string(supportedIncidentTypes[0])
// 	incidentTypeOptions := []*slack.OptionBlockObject{}
// 	for _, incidentType := range supportedIncidentTypes {
// 		incidentTypeOptions = append(incidentTypeOptions, slack.NewOptionBlockObject(
// 			string(incidentType),
// 			slack.NewTextBlockObject(slack.PlainTextType,
// 				string(incidentType), false, false),
// 			slack.NewTextBlockObject(slack.PlainTextType,
// 				string(incidentType), false, false),
// 		))
// 	}

// 	return slack.NewSectionBlock(
// 		&slack.TextBlockObject{
// 			Type: slack.MarkdownType,
// 			Text: ":trackball: *Type of the incident*",
// 		},
// 		nil, slack.NewAccessory(
// 			&slack.SelectBlockElement{
// 				Type:        slack.OptTypeStatic,
// 				ActionID:    "create_incident_modal_set_incident_type",
// 				Placeholder: slack.NewTextBlockObject(slack.PlainTextType, "Select incident type...", false, false),
// 				InitialOption: slack.NewOptionBlockObject(
// 					initialOptionForIncidentType,
// 					slack.NewTextBlockObject(slack.PlainTextType,
// 						initialOptionForIncidentType, false, false),
// 					slack.NewTextBlockObject(slack.PlainTextType,
// 						initialOptionForIncidentType, false, false),
// 				),
// 				Options: incidentTypeOptions,
// 			},
// 		), slack.SectionBlockOptionBlockID("incident_type"),
// 	)
// }

func (is incidentService) CreateIncidentView(evt *socketmode.Event) {
	callback, ok := evt.Data.(slack.InteractionCallback)
	if !ok {
		logrus.Infof("Ignored %+v\n", evt)
		return
	}
	// Send Ack
	is.client.Ack(*evt.Request)

	slackBlocks := []slack.Block{
		slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: "This will start a new incident channel and you will " +
					"be invited to it. From there, please use our incident " +
					"management process to run the incident or coordinate " +
					"with others to do so.",
			},
			nil, nil,
		),
		getNameBlock(),
		getTypeBlock(),
		getSummaryBlock(),
		getSeverityBlock(),
		getRoleBlock(),
		getChannelSelectBlock(),
	}

	viewResp, err := is.client.OpenView(callback.TriggerID, slack.ModalViewRequest{
		Type:       slack.VTModal,
		CallbackID: "create_incident_modal",
		Title:      slack.NewTextBlockObject(slack.PlainTextType, "Start a new incident", false, false),
		Blocks: slack.Blocks{
			BlockSet: slackBlocks,
		},
		Submit: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Start",
		},
	})
	logrus.Infof("resp: %+v, err: %+v", viewResp, err)
}

func (is incidentService) CreateIncident(evt *socketmode.Event) {
	callback, ok := evt.Data.(slack.InteractionCallback)
	if !ok {
		logrus.Infof("Ignored %+v\n", evt)
		return
	}
	// Send Ack
	is.client.Ack(*evt.Request)

	responseSubmitted := callback.View.State.Values
	slackUser := callback.User

	if responseSubmitted == nil {
		logrus.Errorf("failed to process create incident request from slack: %s", "empty response submitted")
		return
	}

	name := responseSubmitted["create_incident_modal_name"]["create_incident_modal_set_name"]
	incidentType := responseSubmitted["incident_type"]["create_incident_modal_set_incident_type"]
	summary := responseSubmitted["create_incident_modal_summary"]["create_incident_modal_set_summary"]
	severity := responseSubmitted["incident_severity"]["create_incident_modal_set_incident_severity"]

	var incidentRoles []incidentdb.Role
	roles := responseSubmitted["incident_role"]["create_incident_modal_set_incident_role"]
	for _, role := range roles.SelectedOptions {
		incidentRoles = append(incidentRoles, incidentdb.Role{
			Type: incidentdb.RoleType(role.Value),
			User: utils.UserDetails{
				Source:   utils.Slack,
				UserId:   slackUser.ID,
				UserName: slackUser.Name,
			},
		})
	}

	selectedChannelForResponse := responseSubmitted["create_incident_modal_conversation_select"]["create_incident_modal_select_conversation"]

	createdAt := time.Now()
	channel, err := is.client.Client.CreateConversation(slack.CreateConversationParams{
		ChannelName: generateSlackChannelName(name.Value, &createdAt),
		IsPrivate:   false,
		TeamID:      callback.Team.ID,
	})
	if err != nil {
		logrus.Errorf("failed to create incident channel: %+v", err)
		return
	} else {
		logrus.Infof("Successfully created an incident channel: %+v", channel)
	}
	_, err = is.client.Client.InviteUsersToConversation(channel.ID, slackUser.ID)
	if err != nil {
		logrus.Errorf("failed to invite user %s to incident channel %s: %+v", slackUser.ID, channel.ID, err)
		return
	} else {
		logrus.Infof("Successfully invited user %s to incident channel %s", slackUser.ID, channel.ID)
	}

	createIncidentReq := incident.CreateRequest{
		ResourceDetails: mongodb.ResourceDetails{
			Name:        name.Value,
			Identifier:  name.Value + "-" + uuid.New().String(),
			Description: summary.Value,
		},
		Type:     incidentdb.Type(incidentType.SelectedOption.Value),
		Summary:  summary.Value,
		Severity: incidentdb.Severity(severity.SelectedOption.Value),
		Roles:    incidentRoles,
		Status:   incidentdb.Started,
		IncidentChannel: &incidentdb.IncidentChannel{
			Type: incidentdb.ChannelSlack,
			Slack: &incidentdb.Slack{
				SlackTeam: incidentdb.SlackTeam{
					TeamID:     callback.Team.ID,
					TeamName:   callback.Team.Name,
					TeamDomain: callback.Team.Domain,
				},
				ChannelID: callback.Team.ID,
			},
		},
		Channels: []incidentdb.Channel{
			{
				ID:     channel.ID,
				Name:   channel.Name,
				Source: incidentdb.SlackSource,
				Status: incidentdb.Operational,
				// URL: channel.,
			},
		},
	}
	user := auth.CurrentUser{
		Name:      slackUser.Name,
		Username:  slackUser.Name,
		AccountID: "default",
	}
	newIncident, err := incident.NewIncidentService(incidentdb.NewIncidentOperator(mongodb.Operator),
		"default", "default", "default").Create(context.TODO(), createIncidentReq, user, "")
	if err != nil {
		logrus.Errorf("failed to create incident: %+v", err)
		return
	}
	logrus.Infof("A new incident has been created: %+v", newIncident)

	err = is.sendCreateIncidentResponseMsg(slackUser.TeamID, selectedChannelForResponse.SelectedChannel,
		channel.Name, newIncident.Incident)
	if err != nil {
		logrus.Errorf("failed to post message to the channel: %s, error: %+v",
			selectedChannelForResponse.SelectedChannel, err)
		return
	} else {
		logrus.Infof("A new incident creation response successfully posted to channel:%s",
			selectedChannelForResponse.SelectedChannel)
	}

	err = is.sendCreateIncidentResponseMsg(slackUser.TeamID, channel.ID, channel.Name, newIncident.Incident)
	if err != nil {
		logrus.Errorf("failed to post message to the channel: %s, error: %+v", channel.ID, err)
		return
	} else {
		logrus.Infof("A new incident creation response successfully posted to channel:%s", channel.ID)
	}
}

func (is incidentService) sendCreateIncidentResponseMsg(teamID, channelID, joinChannelName string,
	newIncident incidentdb.Incident) error {
	var commander string
	for _, role := range newIncident.Roles {
		if role.Type == incidentdb.IncidentCommander {
			commander = role.User.UserName
			break
		}
	}

	blocks := []slack.Block{
		slack.NewHeaderBlock(&slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: ":fire: :mega: New Incident",
		}, slack.HeaderBlockOptionBlockID("create_incident_channel_resp_header")),

		slack.NewSectionBlock(&slack.TextBlockObject{
			Type: slack.MarkdownType,
			Text: fmt.Sprintf(":writing_hand: *Name*\n  _%s_", newIncident.Name),
		}, nil, nil, slack.SectionBlockOptionBlockID("create_incident_channel_resp_name")),

		// slack.NewSectionBlock(&slack.TextBlockObject{
		// 	Type: slack.MarkdownType,
		// 	Text: fmt.Sprintf(":hotsprings: *Type*\n  _%s_", newIncident.Type),
		// }, nil, nil, slack.SectionBlockOptionBlockID("create_incident_channel_resp_type")),

		slack.NewSectionBlock(&slack.TextBlockObject{
			Type: slack.MarkdownType,
			Text: fmt.Sprintf(":open_book: *Summary*\n  _%s_", newIncident.Summary),
		}, nil, nil, slack.SectionBlockOptionBlockID("create_incident_channel_resp_summary")),

		slack.NewSectionBlock(&slack.TextBlockObject{
			Type: slack.MarkdownType,
			Text: fmt.Sprintf(":vertical_traffic_light: *Severity*\n  _%s_", newIncident.Severity),
		}, nil, nil, slack.SectionBlockOptionBlockID("create_incident_channel_resp_severity")),

		slack.NewDividerBlock(),

		slack.NewSectionBlock(&slack.TextBlockObject{
			Type: slack.MarkdownType,
			Text: fmt.Sprintf(":eyes: *Current Status*:  _%s_", newIncident.Status),
		}, nil, nil, slack.SectionBlockOptionBlockID("create_incident_channel_resp_status")),
	}

	if len(commander) > 0 {
		blocks = append(blocks,
			slack.NewSectionBlock(&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: fmt.Sprintf(":firefighter: *Commander*:  _%s_", commander),
			}, nil, nil, slack.SectionBlockOptionBlockID("create_incident_channel_resp_commander")),
		)
	}

	blocks = append(blocks,
		slack.NewDividerBlock(),

		slack.NewActionBlock("create_incident_channel_join_channel", slack.ButtonBlockElement{
			Type: slack.METButton,
			Text: &slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: fmt.Sprintf(":slack: %s", joinChannelName),
			},
			Style:    slack.StylePrimary,
			URL:      fmt.Sprintf("https://%s.slack.com/archives/%s", teamID, channelID),
			ActionID: "create_incident_channel_join_channel_button",
			Value:    channelID,
		}),

		slack.NewContextBlock("create_incident_channel_resp_createdAt", slack.NewTextBlockObject(
			slack.PlainTextType, fmt.Sprintf(":clock1: Started At: %q", newIncident.CreatedAt), false, false,
		)),

		slack.NewContextBlock("create_incident_channel_resp_createdBy", slack.NewTextBlockObject(
			slack.PlainTextType, fmt.Sprintf(":man: Started By: %s", newIncident.CreatedBy.UserName), false, false,
		)),
	)

	_, _, err := is.client.Client.PostMessage(channelID, slack.MsgOptionBlocks(blocks...))
	if err != nil {
		return err
	}

	return nil
}

func generateSlackChannelName(incidentName string, createdAt *time.Time) string {
	fmtDateTime := createdAt.Format("20060102150405")
	incidentName = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(incidentName), " ", "-"))
	incidentName = regexp.MustCompile(`[^a-zA-Z0-9 ]+`).ReplaceAllString(incidentName, "")

	return "rn-" + fmtDateTime + "-" + incidentName
}

func (is incidentService) HandleJoinChannelAction(evt *socketmode.Event, blockAction *slack.BlockAction) {
	callback, ok := evt.Data.(slack.InteractionCallback)
	if !ok {
		logrus.Infof("Ignored %+v\n", evt)
		return
	}
	// Send Ack
	is.client.Ack(*evt.Request)

	_, err := is.client.Client.InviteUsersToConversation(blockAction.Value, callback.User.ID)
	if err != nil {
		logrus.Errorf("failed to invite user %s to incident channel %s: %+v", callback.User.ID,
			blockAction.Value, err)
		return
	} else {
		logrus.Infof("Successfully invited user %s to incident channel %s", callback.User.ID, blockAction.Value)
	}
}
