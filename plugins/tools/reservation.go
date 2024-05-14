package tools

import (
	"github.com/Pishia-IA/core/config"
	log "github.com/sirupsen/logrus"
)

type Reservation struct {
}

func NewReservation(config *config.Base) *Reservation {
	return &Reservation{}
}

func (c *Reservation) Run(params map[string]interface{}, userQuery string) (*ToolResponse, error) {
	log.Debugf("Running the Reservation tool with the following parameters: %v", params)

	return &ToolResponse{
		Success: true,
		Type:    "string",
		Data:    "The reservation call has been made.",
	}, nil
}

func (c *Reservation) Setup() error {
	return nil
}

func (c *Reservation) Description() string {
	return "Making reservation making call"
}

func (c *Reservation) Parameters() map[string]*ToolParameter {
	return map[string]*ToolParameter{
		"phone_number": {
			Description: "The phone number of the person or business to call, to make the reservation, must be provided in the correct format. This can't be equal to the user's phone number.",
			Type:        "string",
			Required:    true,
		},
		"objective": {
			Description: "The specific purpose of the reservation call, such as booking a table or scheduling an appointment, must be clearly defined and non-empty. Include time and date if necessary. All the necessary information to complete the reservation should be included in the objective.",
			Type:        "string",
			Required:    true,
		},
		"id_user_name": {
			Description: "The name of the user for whom the reservation is being made, to be used in the call if necessary.",
			Type:        "string",
			Required:    false,
		},
		"id_user_email": {
			Description: "The email address of the user for whom the reservation is being made, to be used in the call if necessary.",
			Type:        "string",
			Required:    false,
		},
		"id_user_phone": {
			Description: "The phone number of the user for whom the reservation is being made, to be used in the call if necessary.",
			Type:        "string",
			Required:    false,
		},
		"initial_message": {
			Description: "The initial message to be played when the call is answered, to provide context for the reservation. Please, greet the person on the other end, and state the purpose of the call. Be kind and polite. You should generate a message that is appropriate for the context.",
			Type:        "string",
			Required:    true,
		},
	}
}

func (c *Reservation) UseCase() []string {
	return []string{
		"Initiate a reservation call for the user, ensuring the objective is clearly communicated as the interaction involves a human.",
		"Make a restaurant reservation call for the user, clearly stating the purpose of the call to the person on the other end.",
		"Make a hotel reservation call for the user, with a clear and direct objective for the conversation.",
		"Make a service reservation call for the user, ensuring the goal of the call is clear since it will be with a human.",
		"If the user does not provide a phone number in the correct format, request it before executing the tool.",
		"Do not execute this tool without the necessary information, including properly formatted phone numbers.",
		"id_user_name, id_user_email, and id_user_phone are optional parameters that can be used to personalize the call, but one of them must be provided.",
		"If user want to do a reservation for dinner, lunch, or breakfast, the tool should be able to handle it the number of people and the time. Maybe you can understand if dinner, lunch or breakfast based on the time of the day.",
		"If additional information is needed, ask the user whether to retrieve it from search results or to have them provide it directly.",
		"Convert the phone number to an international format if it is not already in the correct format, ensuring it adheres to standard conventions.",
	}
}
