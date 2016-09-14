package permissions

import (
	"github.com/xyproto/pinterface"
	"testing"
)

func TestPerm(t *testing.T) {
	userstate := NewUserStateSimple()

	userstate.AddUser("bob", "hunter1", "bob@zombo.com")

	if !userstate.HasUser("bob") {
		t.Error("Error, user bob should exist")
	}

	if userstate.IsConfirmed("bob") {
		t.Error("Error, user bob should not be confirmed right now.")
	}

	userstate.MarkConfirmed("bob")

	if !userstate.IsConfirmed("bob") {
		t.Error("Error, user bob should be marked as confirmed right now.")
	}

	if userstate.IsAdmin("bob") {
		t.Error("Error, user bob should not have admin rights")
	}

	userstate.SetAdminStatus("bob")

	if !userstate.IsAdmin("bob") {
		t.Error("Error, user bob should have admin rights")
	}

	userstate.RemoveUser("bob")

	if userstate.HasUser("bob") {
		t.Error("Error, user bob should not exist")
	}
}

func TestPasswordBasic(t *testing.T) {
	userstate := NewUserStateSimple()

	// Assert that the default password algorithm is "bcrypt+"
	if userstate.PasswordAlgo() != "bcrypt+" {
		t.Error("Error, bcrypt+ should be the default password algorithm")
	}

	// Set password algorithm
	userstate.SetPasswordAlgo("sha256")

	// Assert that the algorithm is now sha256
	if userstate.PasswordAlgo() != "sha256" {
		t.Error("Error, setting password algorithm failed")
	}

}

// Check if the functionality for backwards compatible hashing works
func TestPasswordBackward(t *testing.T) {
	userstate := NewUserStateSimple()
	userstate.SetPasswordAlgo("sha256")
	userstate.AddUser("bob", "hunter1", "bob@zombo.com")
	if !userstate.HasUser("bob") {
		t.Error("Error, user bob should exist")
	}
	userstate.SetPasswordAlgo("sha256")
	if !userstate.CorrectPassword("bob", "hunter1") {
		t.Error("Error, the sha256 password really is correct")
	}

	userstate.SetPasswordAlgo("bcrypt")
	if userstate.CorrectPassword("bob", "hunter1") {
		t.Error("Error, the password as stored as sha256, not bcrypt")
	}

	userstate.SetPasswordAlgo("bcrypt+")
	if !userstate.CorrectPassword("bob", "hunter1") {
		t.Error("Error, the sha256 password is not correct when checking with bcrypt+")
	}

	userstate.RemoveUser("bob")
}

// Check if the functionality for backwards compatible hashing works
func TestPasswordNotBackward(t *testing.T) {
	userstate := NewUserStateSimple()
	userstate.SetPasswordAlgo("bcrypt")
	userstate.AddUser("bob", "hunter1", "bob@zombo.com")
	if !userstate.HasUser("bob") {
		t.Error("Error, user bob should exist")
	}
	userstate.SetPasswordAlgo("sha256")
	if userstate.CorrectPassword("bob", "hunter1") {
		t.Error("Error, the password is stored as bcrypt, should not be okay with sha256")
	}

	userstate.SetPasswordAlgo("bcrypt")
	if !userstate.CorrectPassword("bob", "hunter1") {
		t.Error("Error, the password should be correct when checking with bcrypt")
	}

	userstate.RemoveUser("bob")
}

func TestPasswordAlgoMatching(t *testing.T) {
	userstate := NewUserStateSimple()
	// generate two different password using the same credentials but different algos
	userstate.SetPasswordAlgo("sha256")
	sha256_hash := userstate.HashPassword("testuser@example.com", "textpassword")
	userstate.SetPasswordAlgo("bcrypt")
	bcrypt_hash := userstate.HashPassword("testuser@example.com", "textpassword")

	// they shouldn't match
	if sha256_hash == bcrypt_hash {
		t.Error("Error, different algorithms should not have a password match")
	}
}

func TestUserStateKeeper(t *testing.T) {
	userstate := NewUserStateSimple()

	// Check that the userstate qualifies for the IUserState interface
	var _ pinterface.IUserState = userstate
}

func TestHostPassword(t *testing.T) {
	//userstate := NewUserStateWithPassword("localhost", "foobared")
	userstate := NewUserStateWithPassword("localhost", "")

	userstate.AddUser("bob", "hunter1", "bob@zombo.com")
	if !userstate.HasUser("bob") {
		t.Error("Error, user bob should exist")
	}

	userstate.RemoveUser("bob")
	if userstate.HasUser("bob") {
		t.Error("Error, user bob should not exist")
	}
}
