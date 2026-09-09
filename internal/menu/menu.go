package menu

import (
	"fmt"
	"io"
	"time"

	"gopkg.in/yaml.v3"
)

type Localized struct {
	DE string `yaml:"de"`
	EN string `yaml:"en"`
}

type Config struct {
	Conference Conference           `yaml:"conference"`
	Tags       map[string]Localized `yaml:"tags"`
	Permanent  Permanent            `yaml:"permanent"`
	Days       []Day                `yaml:"days"`
}

type Conference struct {
	Name     Localized `yaml:"name"`
	Location Localized `yaml:"location"`
}

type Permanent struct {
	Drinks []Item `yaml:"drinks"`
	Snacks []Item `yaml:"snacks"`
}

type Day struct {
	Date     string    `yaml:"date"`
	Services []Service `yaml:"services"`
}

type Service struct {
	ID       string    `yaml:"id"`
	Title    Localized `yaml:"title"`
	Subtitle Localized `yaml:"subtitle"`
	From     string    `yaml:"from"`
	Until    string    `yaml:"until"`
	Items    []Item    `yaml:"items"`
}

type Item struct {
	ID          string      `yaml:"id"`
	Name        Localized   `yaml:"name"`
	Description Localized   `yaml:"description"`
	Variants    []Localized `yaml:"variants"`
	Tags        []string    `yaml:"tags"`
}

func Decode(reader io.Reader) (Config, error) {
	decoder := yaml.NewDecoder(reader)
	decoder.KnownFields(true)

	var config Config
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("decode menu YAML: %w", err)
	}
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (config Config) Validate() error {
	if err := validateLocalized("conference.name", config.Conference.Name); err != nil {
		return err
	}
	if err := validateLocalized("conference.location", config.Conference.Location); err != nil {
		return err
	}
	for id, label := range config.Tags {
		if id == "" {
			return fmt.Errorf("tag ID is required")
		}
		if err := validateLocalized(fmt.Sprintf("tags.%s", id), label); err != nil {
			return err
		}
	}
	for index, item := range config.Permanent.Drinks {
		if err := config.validateItem(fmt.Sprintf("permanent.drinks[%d]", index), item); err != nil {
			return err
		}
	}
	for index, item := range config.Permanent.Snacks {
		if err := config.validateItem(fmt.Sprintf("permanent.snacks[%d]", index), item); err != nil {
			return err
		}
	}
	if len(config.Days) == 0 {
		return fmt.Errorf("menu must contain at least one day")
	}

	seenDays := make(map[string]bool, len(config.Days))
	for dayIndex, day := range config.Days {
		if _, err := time.Parse("2006-01-02", day.Date); err != nil {
			return fmt.Errorf("days[%d].date must use YYYY-MM-DD", dayIndex)
		}
		if seenDays[day.Date] {
			return fmt.Errorf("day %q occurs more than once", day.Date)
		}
		seenDays[day.Date] = true
		if len(day.Services) == 0 {
			return fmt.Errorf("days[%d] must contain at least one service", dayIndex)
		}

		for serviceIndex, service := range day.Services {
			path := fmt.Sprintf("days[%d].services[%d]", dayIndex, serviceIndex)
			if service.ID == "" {
				return fmt.Errorf("%s.id is required", path)
			}
			if err := validateLocalized(path+".title", service.Title); err != nil {
				return err
			}
			if err := validateLocalized(path+".subtitle", service.Subtitle); err != nil {
				return err
			}
			if err := validateTime(path+".from", service.From); err != nil {
				return err
			}
			if err := validateTime(path+".until", service.Until); err != nil {
				return err
			}
			for itemIndex, item := range service.Items {
				if err := config.validateItem(fmt.Sprintf("%s.items[%d]", path, itemIndex), item); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (config Config) validateItem(path string, item Item) error {
	if item.ID == "" {
		return fmt.Errorf("%s.id is required", path)
	}
	if err := validateLocalized(path+".name", item.Name); err != nil {
		return err
	}
	if err := validateOptionalLocalized(path+".description", item.Description); err != nil {
		return err
	}
	for index, variant := range item.Variants {
		if err := validateLocalized(fmt.Sprintf("%s.variants[%d]", path, index), variant); err != nil {
			return err
		}
	}
	for _, tag := range item.Tags {
		if _, ok := config.Tags[tag]; !ok {
			return fmt.Errorf("%s references unknown tag %q", path, tag)
		}
	}
	return nil
}

func validateLocalized(path string, value Localized) error {
	if value.DE == "" || value.EN == "" {
		return fmt.Errorf("%s requires both de and en", path)
	}
	return nil
}

func validateOptionalLocalized(path string, value Localized) error {
	if (value.DE == "") != (value.EN == "") {
		return fmt.Errorf("%s requires both de and en when present", path)
	}
	return nil
}

func validateTime(path, value string) error {
	if _, err := time.Parse("15:04", value); err != nil {
		return fmt.Errorf("%s must use HH:MM", path)
	}
	return nil
}
