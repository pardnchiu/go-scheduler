package goCron

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
	return parseCron(spec)
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

	if len(field.Values) > 0 {
		for _, v := range field.Values {
			if v == value {
				return true
			}
		}
		return false
	}

	return field.Value == value
}

func parseDescriptor(spec string) (schedule, error) {
	switch spec {
	case "@yearly", "@annually":
		return &scheduleResult{
			scheduleField{Value: 0},
			scheduleField{Value: 0},
			scheduleField{Value: 1},
			scheduleField{Value: 1},
			scheduleField{All: true},
		}, nil
	case "@monthly":
		return &scheduleResult{
			scheduleField{Value: 0},
			scheduleField{Value: 0},
			scheduleField{Value: 1},
			scheduleField{All: true},
			scheduleField{All: true},
		}, nil
	case "@weekly":
		return &scheduleResult{
			scheduleField{Value: 0},
			scheduleField{Value: 0},
			scheduleField{All: true},
			scheduleField{All: true},
			scheduleField{Value: 0},
		}, nil
	case "@daily", "@midnight":
		return &scheduleResult{
			scheduleField{Value: 0},
			scheduleField{Value: 0},
			scheduleField{All: true},
			scheduleField{All: true},
			scheduleField{All: true},
		}, nil
	case "@hourly":
		return &scheduleResult{
			scheduleField{Value: 0},
			scheduleField{All: true},
			scheduleField{All: true},
			scheduleField{All: true},
			scheduleField{All: true},
		}, nil
	}

	if strings.HasPrefix(spec, "@every ") {
		duration, err := time.ParseDuration(spec[7:])
		if err != nil {
			return nil, fmt.Errorf("failed to parse @every: %v", err)
		}
		if duration < 30*time.Second {
			return nil, fmt.Errorf("@every minimum interval is 30s, got %v", duration)
		}
		return delayScheduleResult{duration}, nil
	}

	return nil, fmt.Errorf("failed to parse: %s", spec)
}

func parseCron(spec string) (schedule, error) {
	fields := strings.Fields(spec)
	if len(fields) != 5 {
		return nil, fmt.Errorf("requires 5 values, got %d", len(fields))
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
		return scheduleField{All: true}, nil
	}

	if strings.HasPrefix(field, "*/") {
		str := field[2:]
		step, err := strconv.Atoi(str)
		if err != nil {
			return scheduleField{}, fmt.Errorf("invalid step value: %v", err)
		}
		if step <= 0 {
			return scheduleField{}, fmt.Errorf("step must greater than 0, got %d", step)
		}
		return scheduleField{Step: step}, nil
	}

	if strings.Contains(field, ",") {
		return parseList(field, min, max)
	}

	if strings.Contains(field, "-") {
		return parseRange(field, min, max)
	}

	value, err := strconv.Atoi(field)
	if err != nil {
		return scheduleField{}, fmt.Errorf("invalid value: %v", err)
	}

	if value < min || value > max {
		return scheduleField{}, fmt.Errorf("%d out of range [%d, %d]", value, min, max)
	}

	return scheduleField{Value: value}, nil
}

func parseRange(field string, min, max int) (scheduleField, error) {
	if strings.HasPrefix(field, "-") {
		return scheduleField{}, fmt.Errorf("cannot start with %s", field)
	}
	if strings.HasSuffix(field, "-") {
		return scheduleField{}, fmt.Errorf("cannot end with %s", field)
	}
	if strings.Contains(field, "--") {
		return scheduleField{}, fmt.Errorf("cannot contain multiple %s", field)
	}

	parts := strings.Split(field, "-")
	if len(parts) != 2 {
		return scheduleField{}, fmt.Errorf("invalid format: %s", field)
	}

	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return scheduleField{}, fmt.Errorf("invalid start: %v", err)
	}

	end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return scheduleField{}, fmt.Errorf("invalid end: %v", err)
	}

	if start < min || start > max {
		return scheduleField{}, fmt.Errorf("%d out of bounds [%d, %d]", start, min, max)
	}
	if end < min || end > max {
		return scheduleField{}, fmt.Errorf("%d out of bounds [%d, %d]", end, min, max)
	}
	if start > end {
		return scheduleField{}, fmt.Errorf("%d cannot be greater than %d", start, end)
	}

	var values []int
	for i := start; i <= end; i++ {
		values = append(values, i)
	}

	return scheduleField{Values: values}, nil
}

func parseList(field string, min, max int) (scheduleField, error) {
	if strings.HasPrefix(field, ",") {
		return scheduleField{}, fmt.Errorf("cannot start with %s", field)
	}
	if strings.HasSuffix(field, ",") {
		return scheduleField{}, fmt.Errorf("cannot end with %s", field)
	}
	if strings.Contains(field, ",,") {
		return scheduleField{}, fmt.Errorf("cannot contain multiple %s", field)
	}

	parts := strings.Split(field, ",")
	var allValues []int
	valueSet := make(map[int]bool)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return scheduleField{}, fmt.Errorf("empty field %s", field)
		}

		var values []int

		if strings.Contains(part, "-") {
			rangeField, err := parseRange(part, min, max)
			if err != nil {
				return scheduleField{}, fmt.Errorf("invalid range %v", err)
			}
			values = rangeField.Values
		} else {
			value, err := strconv.Atoi(part)
			if err != nil {
				return scheduleField{}, fmt.Errorf("invalid value %v", err)
			}
			if value < min || value > max {
				return scheduleField{}, fmt.Errorf("%d out of range [%d, %d]", value, min, max)
			}
			values = []int{value}
		}

		for _, v := range values {
			if !valueSet[v] {
				valueSet[v] = true
				allValues = append(allValues, v)
			}
		}
	}

	if len(allValues) == 0 {
		return scheduleField{}, fmt.Errorf("empty list field: %s", field)
	}

	return scheduleField{Values: allValues}, nil
}
