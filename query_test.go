package oorm

import (
	"testing"
)

func TestDB_Find(t *testing.T) {
	var u User
	err := orm.NewDB().Where("id", ">", 0).Where("user_name", "kwin").First(&u)

	if err != nil {
		t.Error(err)
	}
	t.Log(u)
}

func TestDB_Get(t *testing.T) {
	var u []User
	err := orm.NewDB().Where("id", ">", 0).Select("user_name", "status as c").Where("user_name", "kwin").Get(&u)

	if err != nil {
		t.Error(err)
	}
	t.Log(u)
}

func TestDB_Value(t *testing.T) {

	var userName string
	err := orm.NewDB().Table("user").Where("id", "=", 1).Value("user_name", &userName)

	if err != nil {
		t.Error(err)
	}
	t.Logf("%#v", userName)
}
