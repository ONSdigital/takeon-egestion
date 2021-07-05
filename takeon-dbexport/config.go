package main

// Config data
type Config struct {
	Reference      string        `json:"reference"`
	Period         string        `json:"period"`
	Survey         string        `json:"survey"`
	Periodicity    string        `json:"periodicity"`
	BpmID          string        `json:"bpmid"`
	Contributors   []Contributor `json:"contributor"`
	Responses      []Response    `json:"response"`
	QuestionSchema []struct {
		Period    string `json:"period"`
		Survey    string `json:"survey"`
		Question  string `json:"question"`
		Datatype  string `json:"datatype"`
		Repeating bool   `json:"repeating"`
	} `json:"question_schema"`
	Validations []struct {
		Rule            string `json:"rule"`
		Formula         string `json:"baseformula"`
		ValidationID    int    `json:"validationid"`
		PrimaryQuestion string `json:"primaryquestion"`
		Default         string `json:"defaultvalue"`
		OutputFormula   string
		Parameters      []ValidationParameter `json:"parameters"`
	} `json:"validation_config"`
	LoggerContext struct {
		Log_level            string `json:"log_level"`
		Log_correlation_id   string `json:"log_correlation_id"`
		Log_correlation_type string `json:"log_correlation_type"`
		Survey               string `json:"survey"`
		Period               string `json:"period"`
		Reference            string `json:"reference"`
	} `json:"context"`
}

// ValidationParameter ...
type ValidationParameter struct {
	Name             string `json:"parameter"`
	Value            string `json:"value"`
	Source           string `json:"source"`
	PeriodOffset     int    `json:"periodoffset"`
	OffsetPeriod     string
	ReplacementValue string
}

// Contributor ...
type Contributor struct {
	Reference string `json:"reference"`
	Period    string `json:"period"`
	Survey    string `json:"survey"`
	Status    string `json:"status"`
	Frosic    string `json:"frozensic"`
	Rusic     string `json:"rusic"`
}

// Response ...
type Response struct {
	Reference          string `json:"reference"`
	Period             string `json:"period"`
	Survey             string `json:"survey"`
	Instance           int    `json:"instance"`
	Question           string `json:"questioncode"`
	Response           string `json:"response"`
	DateaAjustmentFlag bool   `json:"dateadjustment"`
	AdjustedResponse   string `json:"adjustedresponse"`
}
