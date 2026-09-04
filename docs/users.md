```go
// IsBuyer reports whether the user has a buyer role.
func (u *User) IsBuyer() bool {
	return u.Role == "buyer"
}

// IsFarmer reports whether the user has a farmer role.
func (u *User) IsFarmer() bool {
	return u.Role == "farmer"
}

// IsOrganisation reports whether the user has an organisation role.
func (u *User) IsOrganisation() bool {
	return u.Role == "organisation"
}

// IsMinistry reports whether the user has a ministry role.
func (u *User) IsMinistry() bool {
	return u.Role == "ministry"
}
```