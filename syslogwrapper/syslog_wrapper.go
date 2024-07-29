// syslog_wrapper.go
package syslogwrapper

import (
	"fmt"
	"log/syslog"
)

// SyslogWrapper is a concrete implementation of SyslogWrapperInterface
type SyslogWrapper struct {
	logger *syslog.Writer
}

func NewSyslogWrapper(tag string) (*SyslogWrapper, error) {
	logger, err := syslog.New(syslog.LOG_WARNING|syslog.LOG_DAEMON, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize syslog: %v", err)
	}
	return &SyslogWrapper{logger: logger}, nil
}

func (s *SyslogWrapper) Close() {
	if err := s.logger.Close(); err != nil {
		fmt.Printf("Failed to close syslog: %v\n", err)
	}
}

func (s *SyslogWrapper) Warning(message string) {
	if err := s.logger.Warning(message); err != nil {
		fmt.Printf("Failed to write warning to syslog: %v\n", err)
	}
}

func (s *SyslogWrapper) Error(message string) {
	if err := s.logger.Err(message); err != nil {
		fmt.Printf("Failed to write error to syslog: %v\n", err)
	}
}

func (s *SyslogWrapper) Info(message string) {
	if err := s.logger.Info(message); err != nil {
		fmt.Printf("Failed to write info to syslog: %v\n", err)
	}
}

func (s *SyslogWrapper) Debug(message string) {
	if err := s.logger.Debug(message); err != nil {
		fmt.Printf("Failed to write debug to syslog: %v\n", err)
	}
}
