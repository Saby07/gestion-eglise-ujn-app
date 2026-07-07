package models

// RoleKey identifie un rôle applicatif.
type RoleKey string

const (
	RoleSuperAdmin RoleKey = "super_admin"
	RoleAdmin      RoleKey = "admin"
	RoleAccountant RoleKey = "accountant"
	RoleCashier    RoleKey = "cashier"
	RoleStaff      RoleKey = "staff"
)

var AllRoles = []RoleKey{RoleSuperAdmin, RoleAdmin, RoleAccountant, RoleCashier, RoleStaff}

func (r RoleKey) Label() string {
	switch r {
	case RoleSuperAdmin:
		return "Super Admin"
	case RoleAdmin:
		return "Admin"
	case RoleAccountant:
		return "Comptable"
	case RoleCashier:
		return "Caissier"
	case RoleStaff:
		return "Staff"
	default:
		return string(r)
	}
}

type Role struct {
	BaseModel
	Key         RoleKey `gorm:"type:varchar(32);uniqueIndex;not null"`
	Name        string  `gorm:"type:varchar(64);not null"`
	Description string  `gorm:"type:varchar(255)"`
}

func (Role) TableName() string { return "roles" }

type User struct {
	BaseModel
	FirstName    string `gorm:"type:varchar(100);not null"`
	LastName     string `gorm:"type:varchar(100);not null"`
	Email        string `gorm:"type:varchar(255);uniqueIndex;not null"`
	Phone        string `gorm:"type:varchar(32)"`
	PasswordHash string `gorm:"type:varchar(255);not null"`
	IsActive     bool   `gorm:"not null;default:true;index"`
	// Superadmin peut autoriser le comptable à décaisser après validation superadmin.
	CanDisburse bool       `gorm:"not null;default:false"`
	Roles       []UserRole `gorm:"foreignKey:UserID"`
}

func (User) TableName() string { return "users" }

func (u User) FullName() string {
	return u.FirstName + " " + u.LastName
}

type UserRole struct {
	BaseModel
	UserID uint `gorm:"not null;uniqueIndex:idx_user_role"`
	User   User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	RoleID uint `gorm:"not null;uniqueIndex:idx_user_role"`
	Role   Role `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (UserRole) TableName() string { return "user_roles" }

func (u User) HasRole(key RoleKey) bool {
	for _, ur := range u.Roles {
		if ur.Role.Key == key {
			return true
		}
	}
	return false
}

// CanViewAccountBalance indique si le solde des comptes peut être affiché (Comptable, Caissier).
func (u User) CanViewAccountBalance() bool {
	return !u.HasRole(RoleStaff) && !u.HasRole(RoleAdmin) && !u.HasRole(RoleSuperAdmin)
}

func (u User) PrimaryRole() RoleKey {
	order := []RoleKey{RoleSuperAdmin, RoleAdmin, RoleAccountant, RoleCashier, RoleStaff}
	for _, want := range order {
		if u.HasRole(want) {
			return want
		}
	}
	return RoleStaff
}
