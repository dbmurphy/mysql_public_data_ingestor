// syslogwrapper_interface.go
package syslogwrapper

// SyslogWrapperInterface defines methods for syslog logging
type SyslogWrapperInterface interface {
	Close()
	Warning(message string)
	Error(message string)
	Info(message string)
	Debug(message string)
}
