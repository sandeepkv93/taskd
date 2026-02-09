package commands

import (
	"fmt"
	"strings"
)

type Type string

const (
	TypeAdd        Type = "add"
	TypeSnooze     Type = "snooze"
	TypeShow       Type = "show"
	TypeReschedule Type = "reschedule"
)

type ErrorCode string

const (
	ErrCodeEmptyInput      ErrorCode = "empty_input"
	ErrCodeUnknownCommand  ErrorCode = "unknown_command"
	ErrCodeInvalidArgument ErrorCode = "invalid_argument"
	ErrCodeHandlerMissing  ErrorCode = "handler_missing"
)

type CommandError struct {
	Code    ErrorCode
	Message string
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

type AddArgs struct {
	Title string
}

type SnoozeArgs struct {
	Target string
	For    string
}

type ShowArgs struct {
	Subject string
	Tag     string
}

type RescheduleArgs struct {
	Target string
	When   string
}

type Command struct {
	Type       Type
	Raw        string
	Add        *AddArgs
	Snooze     *SnoozeArgs
	Show       *ShowArgs
	Reschedule *RescheduleArgs
}

func Parse(input string) (Command, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return Command{}, &CommandError{Code: ErrCodeEmptyInput, Message: "command is empty"}
	}
	if strings.HasPrefix(raw, "/") {
		raw = strings.TrimSpace(strings.TrimPrefix(raw, "/"))
	}
	if raw == "" {
		return Command{}, &CommandError{Code: ErrCodeEmptyInput, Message: "command is empty"}
	}

	parts := strings.Fields(raw)
	head := strings.ToLower(parts[0])
	args := parts[1:]

	switch Type(head) {
	case TypeAdd:
		return parseAdd(input, args)
	case TypeSnooze:
		return parseSnooze(input, args)
	case TypeShow:
		return parseShow(input, args)
	case TypeReschedule:
		return parseReschedule(input, args)
	default:
		return Command{}, &CommandError{Code: ErrCodeUnknownCommand, Message: fmt.Sprintf("unsupported command: %s", head)}
	}
}

func parseAdd(raw string, args []string) (Command, error) {
	if len(args) == 0 {
		return Command{}, &CommandError{Code: ErrCodeInvalidArgument, Message: "add requires a title"}
	}
	title := strings.TrimSpace(strings.Join(args, " "))
	if title == "" {
		return Command{}, &CommandError{Code: ErrCodeInvalidArgument, Message: "add requires a title"}
	}
	return Command{Type: TypeAdd, Raw: raw, Add: &AddArgs{Title: title}}, nil
}

func parseSnooze(raw string, args []string) (Command, error) {
	if len(args) < 2 {
		return Command{}, &CommandError{Code: ErrCodeInvalidArgument, Message: "snooze requires target and duration"}
	}
	return Command{Type: TypeSnooze, Raw: raw, Snooze: &SnoozeArgs{Target: strings.ToLower(args[0]), For: strings.Join(args[1:], " ")}}, nil
}

func parseShow(raw string, args []string) (Command, error) {
	if len(args) == 0 {
		return Command{}, &CommandError{Code: ErrCodeInvalidArgument, Message: "show requires a subject"}
	}
	subject := strings.ToLower(args[0])
	tag := ""
	for _, arg := range args[1:] {
		if strings.HasPrefix(strings.ToLower(arg), "tag:") {
			tag = strings.TrimSpace(strings.TrimPrefix(arg, "tag:"))
		}
	}
	return Command{Type: TypeShow, Raw: raw, Show: &ShowArgs{Subject: subject, Tag: tag}}, nil
}

func parseReschedule(raw string, args []string) (Command, error) {
	if len(args) < 2 {
		return Command{}, &CommandError{Code: ErrCodeInvalidArgument, Message: "reschedule requires target and time"}
	}
	return Command{Type: TypeReschedule, Raw: raw, Reschedule: &RescheduleArgs{Target: strings.ToLower(args[0]), When: strings.Join(args[1:], " ")}}, nil
}
