// Package acceltest gives tests outside accel the hooks accel no longer
// exports. accel installs them from its init, so a test that imports accel
// can call them; without accel they do nothing.
package acceltest

var resetDefault func()

// SetResetDefault is called by accel to install the reset of its shared
// session.
func SetResetDefault(f func()) { resetDefault = f }

// ResetDefault drops the session accel.Default returns, so the next call
// rediscovers devices.
func ResetDefault() {
	if resetDefault != nil {
		resetDefault()
	}
}
