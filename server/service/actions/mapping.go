package actions

import "NanoKVM-Server/proto"

// Translation between the stored shape and the wire shape, kept apart so the
// stored format can change without the API changing with it.

func toProto(cfg Config) *proto.GetActionsRsp {
	list := make([]proto.ActionInfo, 0, len(cfg.Actions))

	for _, action := range cfg.Actions {
		info := proto.ActionInfo{
			Id:         action.Id,
			Name:       action.Name,
			ShowInMenu: action.ShowInMenu,
			Steps:      make([]proto.ActionStep, 0, len(action.Steps)),
		}

		for _, step := range action.Steps {
			out := proto.ActionStep{Type: step.Type, DelayMs: step.DelayMs}

			if step.GPIO != nil {
				out.GPIO = &proto.ActionStepGPIO{
					Pin:        step.GPIO.Pin,
					Mode:       step.GPIO.Mode,
					DurationMs: step.GPIO.DurationMs,
					ActiveLow:  step.GPIO.ActiveLow,
					// Surfaced so the UI can warn that a pin already has a job,
					// rather than refusing a repurpose the user may well intend.
					ReservedFor: ReservedGPIO(step.GPIO.Pin),
				}
			}
			if step.PWM != nil {
				out.PWM = &proto.ActionStepPWM{
					Chip:        step.PWM.Chip,
					Channel:     step.PWM.Channel,
					PeriodNs:    step.PWM.PeriodNs,
					DutyPercent: step.PWM.DutyPercent,
					Enable:      step.PWM.Enable,
				}
			}
			if step.Command != nil {
				out.Command = &proto.ActionStepCommand{
					Command:    step.Command.Command,
					Background: step.Command.Background,
					TimeoutSec: step.Command.TimeoutSec,
				}
			}

			info.Steps = append(info.Steps, out)
		}

		list = append(list, info)
	}

	return &proto.GetActionsRsp{
		Actions:         list,
		ShortPress:      cfg.Buttons.ShortPress,
		DoublePress:     cfg.Buttons.DoublePress,
		LongPress:       cfg.Buttons.LongPress,
		VeryLongPress:   cfg.Buttons.VeryLongPress,
		KeepDefaults:    cfg.Buttons.KeepDefaults,
		MinGPIO:         minGPIO,
		MaxGPIO:         maxGPIO,
		LongPressMs:     longPressMs,
		VeryLongPressMs: veryLongPressMs,
	}
}

func fromProto(req proto.SetActionsReq) Config {
	cfg := Config{
		Actions: make([]Action, 0, len(req.Actions)),
		Buttons: ButtonMap{
			ShortPress:    req.ShortPress,
			DoublePress:   req.DoublePress,
			LongPress:     req.LongPress,
			VeryLongPress: req.VeryLongPress,
			KeepDefaults:  req.KeepDefaults,
		},
	}

	for _, info := range req.Actions {
		action := Action{
			Id:         info.Id,
			Name:       info.Name,
			ShowInMenu: info.ShowInMenu,
			Steps:      make([]Step, 0, len(info.Steps)),
		}

		for _, step := range info.Steps {
			out := Step{Type: step.Type, DelayMs: step.DelayMs}

			if step.GPIO != nil {
				out.GPIO = &StepGPIO{
					Pin:        step.GPIO.Pin,
					Mode:       step.GPIO.Mode,
					DurationMs: step.GPIO.DurationMs,
					ActiveLow:  step.GPIO.ActiveLow,
				}
			}
			if step.PWM != nil {
				out.PWM = &StepPWM{
					Chip:        step.PWM.Chip,
					Channel:     step.PWM.Channel,
					PeriodNs:    step.PWM.PeriodNs,
					DutyPercent: step.PWM.DutyPercent,
					Enable:      step.PWM.Enable,
				}
			}
			if step.Command != nil {
				out.Command = &StepCommand{
					Command:    step.Command.Command,
					Background: step.Command.Background,
					TimeoutSec: step.Command.TimeoutSec,
				}
			}

			action.Steps = append(action.Steps, out)
		}

		cfg.Actions = append(cfg.Actions, action)
	}

	return cfg
}
