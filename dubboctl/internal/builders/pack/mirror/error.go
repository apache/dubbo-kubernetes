package mirror

import (
	"fmt"
)

type ImageNotFoundError string

func (e ImageNotFoundError) Error() string {
	return fmt.Sprintf("pack mirror: image %q not found, please visit https://docker.aityp.com/manage/add to add it", string(e))
}

type ErrLatestTagNotSupported string

func (e ErrLatestTagNotSupported) Error() string {
	return fmt.Sprintf("pack mirror: image %q with latest tag is not supported, please specify a specific tag", string(e))
}

type ErrDigestNotSupported string

func (e ErrDigestNotSupported) Error() string {
	return fmt.Sprintf("pack mirror: image %q with digest is not supported, please specify a specific tag", string(e))
}
