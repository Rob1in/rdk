package imu

import (
	"context"
	"testing"

	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/sensor"
	"go.viam.com/rdk/spatialmath"

	"go.viam.com/test"
)

func TestIMUName(t *testing.T) {
	for _, tc := range []struct {
		TestName string
		Name     string
		Expected resource.Name
	}{
		{
			"missing name",
			"",
			resource.Name{
				UUID: "053e1e0c-20de-59e7-bace-922cb1ada629",
				Subtype: resource.Subtype{
					Type:            resource.Type{Namespace: resource.ResourceNamespaceRDK, ResourceType: resource.ResourceTypeComponent},
					ResourceSubtype: SubtypeName,
				},
				Name: "",
			},
		},
		{
			"all fields included",
			"imu1",
			resource.Name{
				UUID: "aed67198-6075-5806-837a-6d33ee4b5a42",
				Subtype: resource.Subtype{
					Type:            resource.Type{Namespace: resource.ResourceNamespaceRDK, ResourceType: resource.ResourceTypeComponent},
					ResourceSubtype: SubtypeName,
				},
				Name: "imu1",
			},
		},
	} {
		t.Run(tc.TestName, func(t *testing.T) {
			observed := Named(tc.Name)
			test.That(t, observed, test.ShouldResemble, tc.Expected)
		})
	}
}

var (
	av   = spatialmath.AngularVelocity{X: 1, Y: 2, Z: 3}
	ea   = &spatialmath.EulerAngles{Roll: 4, Pitch: 5, Yaw: 6}
	desc = sensor.Description{sensor.Type("imu"), ""}
)

func TestWrapWithReconfigurable(t *testing.T) {
	var actualIMU1 IMU = &mock{Name: "imu1"}
	fakeIMU1, err := WrapWithReconfigurable(actualIMU1)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, fakeIMU1.(*reconfigurableIMU).actual, test.ShouldEqual, actualIMU1)

	_, err = WrapWithReconfigurable(nil)
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "expected resource")

	fakeIMU2, err := WrapWithReconfigurable(fakeIMU1)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, fakeIMU2, test.ShouldEqual, fakeIMU1)
}

func TestReconfigurableIMU(t *testing.T) {
	actualIMU1 := &mock{Name: "imu1"}
	fakeIMU1, err := WrapWithReconfigurable(actualIMU1)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, fakeIMU1.(*reconfigurableIMU).actual, test.ShouldEqual, actualIMU1)

	actualIMU2 := &mock{Name: "imu2"}
	fakeIMU2, err := WrapWithReconfigurable(actualIMU2)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, actualIMU1.reconfCalls, test.ShouldEqual, 0)

	err = fakeIMU1.(*reconfigurableIMU).Reconfigure(fakeIMU2)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, fakeIMU1.(*reconfigurableIMU).actual, test.ShouldEqual, actualIMU2)
	test.That(t, actualIMU1.reconfCalls, test.ShouldEqual, 1)

	err = fakeIMU1.(*reconfigurableIMU).Reconfigure(nil)
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "expected new IMU")
}

func TestAngularVelocity(t *testing.T) {
	actualIMU1 := &mock{Name: "imu1"}
	fakeIMU1, _ := WrapWithReconfigurable(actualIMU1)

	test.That(t, actualIMU1.angularVelocityCalls, test.ShouldEqual, 0)
	vel, err := fakeIMU1.(*reconfigurableIMU).AngularVelocity(context.Background())
	test.That(t, err, test.ShouldBeNil)
	test.That(t, vel, test.ShouldResemble, spatialmath.AngularVelocity{X: 1, Y: 2, Z: 3})
	test.That(t, actualIMU1.angularVelocityCalls, test.ShouldEqual, 1)
}

func TestOrientiation(t *testing.T) {
	actualIMU1 := &mock{Name: "imu1"}
	fakeIMU1, _ := WrapWithReconfigurable(actualIMU1)

	test.That(t, actualIMU1.orientationCalls, test.ShouldEqual, 0)
	angles, err := fakeIMU1.(*reconfigurableIMU).Orientation(context.Background())
	test.That(t, err, test.ShouldBeNil)
	test.That(t, angles, test.ShouldResemble, &spatialmath.EulerAngles{Roll: 4, Pitch: 5, Yaw: 6})
	test.That(t, actualIMU1.orientationCalls, test.ShouldEqual, 1)
}

func TestReadings(t *testing.T) {
	actualIMU1 := &mock{Name: "imu1"}
	fakeIMU1, _ := WrapWithReconfigurable(actualIMU1)

	test.That(t, actualIMU1.readingsCalls, test.ShouldEqual, 0)
	result, err := fakeIMU1.(*reconfigurableIMU).Readings(context.Background())
	test.That(t, err, test.ShouldBeNil)
	test.That(t, result, test.ShouldResemble, []interface{}{av, ea})
	test.That(t, actualIMU1.readingsCalls, test.ShouldEqual, 1)
}

func TestDesc(t *testing.T) {
	actualIMU1 := &mock{Name: "imu1"}
	fakeIMU1, _ := WrapWithReconfigurable(actualIMU1)

	test.That(t, actualIMU1.descCalls, test.ShouldEqual, 0)
	result := fakeIMU1.(*reconfigurableIMU).Desc()
	test.That(t, result, test.ShouldResemble, desc)
	test.That(t, actualIMU1.descCalls, test.ShouldEqual, 1)
}

type mock struct {
	IMU
	Name                 string
	angularVelocityCalls int
	orientationCalls     int
	readingsCalls        int
	descCalls            int
	reconfCalls          int
}

func (m *mock) AngularVelocity(ctx context.Context) (spatialmath.AngularVelocity, error) {
	m.angularVelocityCalls++
	return av, nil
}
func (m *mock) Orientation(ctx context.Context) (spatialmath.Orientation, error) {
	m.orientationCalls++
	return ea, nil
}
func (m *mock) Readings(ctx context.Context) ([]interface{}, error) {
	m.readingsCalls++
	return []interface{}{av, ea}, nil
}
func (m *mock) Desc() sensor.Description {
	m.descCalls++
	return desc
}
func (m *mock) Close() error { m.reconfCalls++; return nil }
