package actions

import "NanoKVM-Server/proto"

// Translation between the stored shape and the wire shape. Kept apart so the
// stored format can change without the API changing with it.

func toProto(cfg Config) *proto.GetActionsRsp {
	list := make([]proto.ActionInfo, 0, len(cfg.Actions))

	for _, action := range cfg.Actions {
		info := proto.ActionInfo{
			Id:         action.Id,
			Name:       action.Name,
			Type:       action.Type,
			Command:    action.Command,
			ShowInMenu: action.ShowInMenu,
		}
		if action.GPIO != nil {
			info.GPIO = &proto.ActionGPIO{
				Pin:        action.GPIO.Pin,
				Mode:       action.GPIO.Mode,
				DurationMs: action.GPIO.DurationMs,
				ActiveLow:  action.GPIO.ActiveLow,
				// Surfaced so the UI can warn that a pin already has a job,
				// rather than refusing a repurpose the user may well intend.
				ReservedFor: ReservedGPIO(action.GPIO.Pin),
			}
		}
		list = append(list, info)
	}

	return &proto.GetActionsRsp{
		Actions:      list,
		ShortPress:   cfg.Buttons.ShortPress,
		LongPress:    cfg.Buttons.LongPress,
		KeepDefaults: cfg.Buttons.KeepDefaults,
		MinGPIO:      minGPIO,
		MaxGPIO:      maxGPIO,
	}
}

func fromProto(req proto.SetActionsReq) Config {
	cfg := Config{
		Actions: make([]Action, 0, len(req.Actions)),
		Buttons: ButtonMap{
			ShortPress:   req.ShortPress,
			LongPress:    req.LongPress,
			KeepDefaults: req.KeepDefaults,
		},
	}

	for _, info := range req.Actions {
		action := Action{
			Id:         info.Id,
			Name:       info.Name,
			Type:       info.Type,
			Command:    info.Command,
			ShowInMenu: info.ShowInMenu,
		}
		if info.GPIO != nil {
			action.GPIO = &GPIOSpec{
				Pin:        info.GPIO.Pin,
				Mode:       info.GPIO.Mode,
				DurationMs: info.GPIO.DurationMs,
				ActiveLow:  info.GPIO.ActiveLow,
			}
		}
		cfg.Actions = append(cfg.Actions, action)
	}

	return cfg
}
