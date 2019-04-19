package polochon

import (
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
)

// Make sure that the test module is a detailer
var _ Detailer = (*testModule)(nil)

type testModule struct {
	name string
}

func (m *testModule) Init([]byte) error {
	return nil
}

func (m *testModule) Name() string {
	return m.name
}

func (m *testModule) Status() (ModuleStatus, error) {
	return StatusOK, nil
}

func (m *testModule) GetDetails(i interface{}, log *logrus.Entry) error {
	return nil
}

func TestModuleRegisterWithoutName(t *testing.T) {
	clearRegisteredModules()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("a module without name should not be able to register")
		}
	}()
	RegisterModule(&testModule{})
}

func TestModuleRegisterWithSameNames(t *testing.T) {
	clearRegisteredModules()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("a module without name should not be able to register")
		}
	}()

	module1 := &testModule{name: "testModule"}
	module2 := &testModule{name: "testModule"}
	RegisterModule(module1)
	RegisterModule(module2)
}

func TestModuleNotFound(t *testing.T) {
	_, err := GetModule("moduleNotFound", TypeCalendar)
	if err == nil {
		t.Fatalf("should get a module not found")
	}
}

func TestModuleTypeNotFound(t *testing.T) {
	clearRegisteredModules()
	moduleName := "test"
	RegisterModule(&testModule{name: moduleName})
	testType := ModuleType("fakeType")
	_, err := GetModule(moduleName, testType)
	if err == nil {
		t.Fatalf("should get a module not found")
	}
}

func TestModuleWithInvalidType(t *testing.T) {
	clearRegisteredModules()
	moduleName := "test"
	RegisterModule(&testModule{name: moduleName})
	_, err := GetModule(moduleName, TypeCalendar)
	if err == nil {
		t.Fatalf("should not be able to get a module with the wrong type")
	}
}

func TestModuleGet(t *testing.T) {
	clearRegisteredModules()
	module := &testModule{name: "test"}
	RegisterModule(module)
	got, err := GetModule(module.name, TypeDetailer)
	if err != nil {
		t.Fatalf("expected nil, got %q", err)
	}

	if !reflect.DeepEqual(got, module) {
		t.Fatalf("should get the right module")
	}
}
