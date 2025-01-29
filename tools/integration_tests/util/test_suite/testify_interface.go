package test_suite

import "github.com/stretchr/testify/suite"

type TestifySuite interface {
	suite.TestingSuite
	Run(name string, subtest func()) bool
}
