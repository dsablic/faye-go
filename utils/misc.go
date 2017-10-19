package utils

func CompareStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type StringSet struct {
	values map[string]bool
}

func NewStringSet() *StringSet {
	return &StringSet{make(map[string]bool)}
}

func (c *StringSet) Add(value string) {
	c.values[value] = true
}

func (c *StringSet) AddMany(values []string) {
	for _, value := range values {
		c.values[value] = true
	}
}

func (c *StringSet) Remove(value string) {
	delete(c.values, value)
}

func (c *StringSet) Has(value string) bool {
	_, ok := c.values[value]
	return ok
}

func (c *StringSet) GetAll() []string {
	all := make([]string, len(c.values))
	i := 0
	for k := range c.values {
		all[i] = k
		i = i + 1
	}
	return all
}

func CompareValueSlices(a, b []interface{}) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type ValueSet struct {
	values map[interface{}]bool
}

func NewValueSet() *ValueSet {
	return &ValueSet{make(map[interface{}]bool)}
}

func (c *ValueSet) Add(value interface{}) {
	c.values[value] = true
}

func (c *ValueSet) AddMany(values []interface{}) {
	for _, value := range values {
		c.values[value] = true
	}
}

func (c *ValueSet) Remove(value interface{}) {
	delete(c.values, value)
}

func (c *ValueSet) Has(value interface{}) bool {
	_, ok := c.values[value]
	return ok
}

func (c *ValueSet) GetAll() []interface{} {
	all := make([]interface{}, len(c.values))
	i := 0
	for k := range c.values {
		all[i] = k
		i = i + 1
	}
	return all
}
