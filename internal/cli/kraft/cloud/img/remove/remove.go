// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2023, Unikraft GmbH and The KraftKit Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package remove

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	kraftcloud "sdk.kraft.cloud"
	kraftcloudimages "sdk.kraft.cloud/images"

	"kraftkit.sh/cmdfactory"
	"kraftkit.sh/config"
	"kraftkit.sh/log"
)

type RemoveOptions struct {
	All    bool                           `long:"all" usage:"Remove all images"`
	Auth   *config.AuthConfig             `noattribute:"true"`
	Client kraftcloudimages.ImagesService `noattribute:"true"`
	Metro  string                         `noattribute:"true"`
}

func NewCmd() *cobra.Command {
	cmd, err := cmdfactory.New(&RemoveOptions{}, cobra.Command{
		Short:   "Delete an image",
		Use:     "rm [FLAGS] NAME[:latest|@sha256:...]",
		Aliases: []string{"delete", "del", "remove"},
		Annotations: map[string]string{
			cmdfactory.AnnotationHelpGroup: "kraftcloud-img",
		},
	})
	if err != nil {
		panic(err)
	}

	return cmd
}

func (opts *RemoveOptions) Pre(cmd *cobra.Command, args []string) error {
	if !opts.All && len(args) == 0 {
		return fmt.Errorf("either specify an image name, or use the --all flag")
	}

	opts.Metro = cmd.Flag("metro").Value.String()
	if opts.Metro == "" {
		opts.Metro = os.Getenv("KRAFTCLOUD_METRO")
	}
	if opts.Metro == "" {
		return fmt.Errorf("kraftcloud metro is unset")
	}

	log.G(cmd.Context()).WithField("metro", opts.Metro).Debug("using")

	return nil
}

func (opts *RemoveOptions) Run(ctx context.Context, args []string) error {
	var err error

	if opts.Auth == nil {
		opts.Auth, err = config.GetKraftCloudAuthConfigFromContext(ctx)
		if err != nil {
			return fmt.Errorf("could not retrieve credentials: %w", err)
		}
	}

	if opts.Client == nil {
		opts.Client = kraftcloud.NewImagesClient(
			kraftcloud.WithToken(config.GetKraftCloudTokenAuthConfig(*opts.Auth)),
		)
	}

	if opts.All {
		images, err := opts.Client.WithMetro(opts.Metro).List(ctx)
		if err != nil {
			return fmt.Errorf("could not get list of all images: %w", err)
		}

		for _, image := range images {
			if !strings.HasPrefix(image.Digest, strings.TrimSuffix(strings.TrimPrefix(opts.Auth.User, "robot$"), ".users.kraftcloud")) {
				continue
			}

			log.G(ctx).Infof("removing %s", image.Digest)

			if err := opts.Client.WithMetro(opts.Metro).DeleteByName(ctx, image.Digest); err != nil {
				log.G(ctx).Errorf("could not delete image: %s", err.Error())
			}
		}
	}

	for _, arg := range args {
		if err := opts.Client.WithMetro(opts.Metro).DeleteByName(ctx, arg); err != nil {
			return fmt.Errorf("could not delete image: %w", err)
		}

		log.G(ctx).Infof("removing %s", arg)
	}

	return err
}