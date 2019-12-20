package opts

import (
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

// ExtraStorageOpt defines a value of a extra storage
type ExtraStorageOpt struct {
	name  string
	value *types.ExtraStorageOptions
}

var _ pflag.Value = &ExtraStorageOpt{}

func NewExtraStorageOpt(name string, ref *types.ExtraStorageOptions) *ExtraStorageOpt {
	return &ExtraStorageOpt{name: name, value: ref}
}

func (opt *ExtraStorageOpt) Name() string {
	return opt.name
}

func (opt *ExtraStorageOpt) Type() string {
	return "extra-storage"
}

func (opt *ExtraStorageOpt) Set(val string) error {
	parts := strings.Split(val, ",")
	if len(parts) != 3 {
		return errors.New("the number of params is not 3")
	}
	opt.value.ExtraPath, opt.value.MountSrcDev, opt.value.DevType = parts[0], parts[1], parts[2]
	return nil
}

func (opt *ExtraStorageOpt) String() string {
	return fmt.Sprintf("extra-dir:%s,mount-src-dev:%s,mount-type:%s", opt.value.ExtraPath, opt.value.MountSrcDev, opt.value.DevType)
}
