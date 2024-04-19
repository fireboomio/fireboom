// Package configs
/*
 重写日志中间件的hooked，辅助实现logger_collector的功能
*/
package configs

import (
	"go.uber.org/multierr"
	"go.uber.org/zap/zapcore"
)

type hooked struct {
	zapcore.Core
	funcs []func(zapcore.Entry, []zapcore.Field) error
}

func registerHooks(core zapcore.Core, hooks ...func(zapcore.Entry, []zapcore.Field) error) zapcore.Core {
	return &hooked{Core: core, funcs: hooks}
}

func (h *hooked) Level() zapcore.Level {
	return zapcore.LevelOf(h.Core)
}

func (h *hooked) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if downstream := h.Core.Check(ent, ce); downstream != nil {
		return downstream.AddCore(ent, h)
	}
	return ce
}

func (h *hooked) With(fields []zapcore.Field) zapcore.Core {
	return &hooked{
		Core:  h.Core.With(fields),
		funcs: h.funcs,
	}
}

func (h *hooked) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	var err error
	for i := range h.funcs {
		err = multierr.Append(err, h.funcs[i](ent, fields))
	}
	return err
}
