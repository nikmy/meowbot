package environment

type Env int

const (
	Unknown Env = iota
	Development
	Production
)

func FromString(s string) Env {
	switch s {
	case "dev":
		return Development
	case "prod":
		return Production
	default:
		return Unknown
	}
}

func (e *Env) UnmarshalYAML(unmarshal func(any) error) error {
	var raw string

	err := unmarshal(&raw)
	if err != nil {
		return err
	}

	*e = FromString(raw)
	return nil
}
