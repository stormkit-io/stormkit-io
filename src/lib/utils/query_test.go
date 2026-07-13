package utils_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type Student struct {
	Name    string `db:"name"`
	Surname string `db:"surname"`
}

func (Student) TableName() string {
	return "students"
}

type QueryGeneratorSuite struct {
	suite.Suite
}

func (s *QueryGeneratorSuite) TestSelectAllExceptStatementGenerator() {
	expected := "SELECT surname FROM students;"
	res := utils.QSelectAllExcept(Student{}, utils.NoFilter, "name")
	s.Equal(expected, res)
}

func (s *QueryGeneratorSuite) TestSelectAllExceptStatementGeneratorWithFilter() {
	expected := "SELECT name FROM students WHERE name='savas';"
	res := utils.QSelectAllExcept(Student{}, `name='savas'`, "surname")
	s.Equal(expected, res)
}

func (s *QueryGeneratorSuite) TestSelectQueryGenerator() {
	expected := "SELECT name,surname FROM students;"
	res := utils.QSelect("students", utils.NoFilter, "name", "surname")
	s.Equal(expected, res)
}

func (s *QueryGeneratorSuite) TestSelectQueryGeneratorWithFilter() {
	expected := "SELECT name,surname FROM students WHERE name='savas';"
	res := utils.QSelect("students", `name='savas'`, "name", "surname")
	s.Equal(expected, res)
}

func (s *QueryGeneratorSuite) TestInsertQueryGenerator() {
	expected := "INSERT INTO students (name, surname) VALUES ($1, $2);"
	res := utils.QInsert("students", "name", "surname")
	s.Equal(expected, res)

	expected = "INSERT INTO students (name) VALUES ($1);"
	res = utils.QInsert("students", "name")
	s.Equal(expected, res)
}

func (s *QueryGeneratorSuite) TestUpdateQueryGenerator() {
	expected := "UPDATE students SET name = $1, surname = $2;"
	res := utils.QUpdate("students", utils.NoFilter, "name", "surname")
	s.Equal(expected, res)
}

func (s *QueryGeneratorSuite) TestUpdateQueryGeneratorWithFilter() {
	expected := "UPDATE students SET name = $1, surname = $2 WHERE name='sa';"
	res := utils.QUpdate("students", `name='sa'`, "name", "surname")
	s.Equal(expected, res)
}

func TestQuery(t *testing.T) {
	suite.Run(t, &QueryGeneratorSuite{})
}
