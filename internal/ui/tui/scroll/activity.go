package scroll

// Profile returns the terminal scroll profile.
func (c *Controller) Profile() Profile {
	return c.profile
}
func (c *Controller) ScrollActive() bool {
	return c.Active
}

// BeginDrain marks drain as active (idempotent).
func (c *Controller) BeginDrain() {
	c.Active = true
}
