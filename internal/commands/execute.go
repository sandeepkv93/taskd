package commands

import "fmt"

type Result struct {
	Message string
}

type Handlers struct {
	Add        func(AddArgs) (Result, error)
	Snooze     func(SnoozeArgs) (Result, error)
	Show       func(ShowArgs) (Result, error)
	Reschedule func(RescheduleArgs) (Result, error)
}

func Execute(cmd Command, handlers Handlers) (Result, error) {
	switch cmd.Type {
	case TypeAdd:
		if handlers.Add == nil {
			return Result{}, &CommandError{Code: ErrCodeHandlerMissing, Message: "add handler not configured"}
		}
		return handlers.Add(*cmd.Add)
	case TypeSnooze:
		if handlers.Snooze == nil {
			return Result{}, &CommandError{Code: ErrCodeHandlerMissing, Message: "snooze handler not configured"}
		}
		return handlers.Snooze(*cmd.Snooze)
	case TypeShow:
		if handlers.Show == nil {
			return Result{}, &CommandError{Code: ErrCodeHandlerMissing, Message: "show handler not configured"}
		}
		return handlers.Show(*cmd.Show)
	case TypeReschedule:
		if handlers.Reschedule == nil {
			return Result{}, &CommandError{Code: ErrCodeHandlerMissing, Message: "reschedule handler not configured"}
		}
		return handlers.Reschedule(*cmd.Reschedule)
	default:
		return Result{}, &CommandError{Code: ErrCodeUnknownCommand, Message: fmt.Sprintf("unknown command type: %s", cmd.Type)}
	}
}
