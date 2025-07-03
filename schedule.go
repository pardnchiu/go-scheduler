package cron

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (parser) parse(spec string) (schedule, error) {
	if spec[0] == '@' {
		return parseDescriptor(spec)
	}
	return parseCronSpec(spec)
}

func (r delayScheduleResult) next(t time.Time) time.Time {
	return t.Add(r.delay)
}

func (s *scheduleResult) next(t time.Time) time.Time {
	t = t.Add(time.Minute - time.Duration(t.Second())*time.Second)

	for {
		if s.matchTime(t) {
			return t
		}
		t = t.Add(time.Minute)
	}
}

func (s *scheduleResult) matchTime(t time.Time) bool {
	return s.matchField(s.minute, t.Minute()) &&
		s.matchField(s.hour, t.Hour()) &&
		s.matchField(s.dom, t.Day()) &&
		s.matchField(s.month, int(t.Month())) &&
		s.matchField(s.dow, int(t.Weekday()))
}

func (s *scheduleResult) matchField(field scheduleField, value int) bool {
	if field.All {
		return true
	}

	if field.Step > 0 {
		return value%field.Step == 0
	}

	return field.Value == value
}

func parseDescriptor(spec string) (schedule, error) {
	switch spec {
	case "@yearly", "@annually":
		return &scheduleResult{
			scheduleField{0, false, 0}, // 分鐘：0
			scheduleField{0, false, 0}, // 小時：0
			scheduleField{1, false, 0}, // 日：1
			scheduleField{1, false, 0}, // 月：1
			scheduleField{0, true, 0},  // 星期：*
		}, nil
	case "@monthly":
		return &scheduleResult{
			scheduleField{0, false, 0}, // 分鐘：0
			scheduleField{0, false, 0}, // 小時：0
			scheduleField{1, false, 0}, // 日：1
			scheduleField{0, true, 0},  // 月：*
			scheduleField{0, true, 0},  // 星期：*
		}, nil
	case "@weekly":
		return &scheduleResult{
			scheduleField{0, false, 0}, // 分鐘：0
			scheduleField{0, false, 0}, // 小時：0
			scheduleField{0, true, 0},  // 日：*
			scheduleField{0, true, 0},  // 月：*
			scheduleField{0, false, 0}, // 星期：0
		}, nil
	case "@daily", "@midnight":
		return &scheduleResult{
			scheduleField{0, false, 0}, // 分鐘：0
			scheduleField{0, false, 0}, // 小時：0
			scheduleField{0, true, 0},  // 日：*
			scheduleField{0, true, 0},  // 月：*
			scheduleField{0, true, 0},  // 星期：*
		}, nil
	case "@hourly":
		return &scheduleResult{
			scheduleField{0, false, 0}, // 分鐘：0
			scheduleField{0, true, 0},  // 小時：*
			scheduleField{0, true, 0},  // 日：*
			scheduleField{0, true, 0},  // 月：*
			scheduleField{0, true, 0},  // 星期：*
		}, nil
	}

	if strings.HasPrefix(spec, "@every ") {
		duration, err := time.ParseDuration(spec[7:])
		if err != nil {
			return nil, fmt.Errorf("Failed to parse @every: %v", err)
		}
		return delayScheduleResult{duration}, nil
	}

	return nil, fmt.Errorf("Failed to parse: %s", spec)
}

func parseCronSpec(spec string) (schedule, error) {
	fields := strings.Fields(spec)
	if len(fields) != 5 {
		return nil, fmt.Errorf("Requires 5 values, got %d", len(fields))
	}

	schedule := &scheduleResult{}
	var err error

	if schedule.minute, err = parseField(fields[0], 0, 59); err != nil {
		return nil, err
	}
	if schedule.hour, err = parseField(fields[1], 0, 23); err != nil {
		return nil, err
	}
	if schedule.dom, err = parseField(fields[2], 1, 31); err != nil {
		return nil, err
	}
	if schedule.month, err = parseField(fields[3], 1, 12); err != nil {
		return nil, err
	}
	if schedule.dow, err = parseField(fields[4], 0, 6); err != nil {
		return nil, err
	}

	return schedule, nil
}

func parseField(field string, min, max int) (scheduleField, error) {
	if field == "*" {
		return scheduleField{0, true, 0}, nil
	}

	if strings.HasPrefix(field, "*/") {
		stepStr := field[2:]
		step, err := strconv.Atoi(stepStr)
		if err != nil {
			return scheduleField{}, err
		}
		return scheduleField{0, false, step}, nil
	}

	value, err := strconv.Atoi(field)
	if err != nil {
		return scheduleField{}, err
	}

	if value < min || value > max {
		return scheduleField{}, fmt.Errorf("Out of range [%d, %d], got [%d]", min, max, value)
	}

	return scheduleField{value, false, 0}, nil
}
