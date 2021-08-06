/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2021 Red Hat, Inc.
 */

package commands

import (
	"log"

	"github.com/fromanirh/deployer/pkg/deployer/api"
	"github.com/fromanirh/deployer/pkg/deployer/rte"
	"github.com/fromanirh/deployer/pkg/deployer/sched"

	"github.com/spf13/cobra"
)

type logAdapter struct {
	log      *log.Logger
	debugLog *log.Logger
}

func (la logAdapter) Printf(format string, v ...interface{}) {
	la.log.Printf(format, v...)
}

func (la logAdapter) Debugf(format string, v ...interface{}) {
	la.debugLog.Printf(format, v...)
}

type deployOptions struct {
	waitCompletion bool
}

func NewDeployCommand(commonOpts *CommonOptions) *cobra.Command {
	opts := &deployOptions{}
	deploy := &cobra.Command{
		Use:   "deploy",
		Short: "deploy the components and configurations needed for topology-aware-scheduling",
		RunE: func(cmd *cobra.Command, args []string) error {
			return deployOnCluster(commonOpts, opts)
		},
		Args: cobra.NoArgs,
	}
	deploy.PersistentFlags().BoolVarP(&opts.waitCompletion, "wait", "W", false, "wait for deployment to be all completed.")
	deploy.AddCommand(NewDeployAPICommand(commonOpts, opts))
	deploy.AddCommand(NewDeploySchedulerPluginCommand(commonOpts, opts))
	deploy.AddCommand(NewDeployTopologyUpdaterCommand(commonOpts, opts))
	return deploy
}

func NewRemoveCommand(commonOpts *CommonOptions) *cobra.Command {
	opts := &deployOptions{}
	remove := &cobra.Command{
		Use:   "remove",
		Short: "remove the components and configurations needed for topology-aware-scheduling",
		RunE: func(cmd *cobra.Command, args []string) error {
			la := logAdapter{
				log:      commonOpts.Log,
				debugLog: commonOpts.DebugLog,
			}
			var err error
			err = sched.Remove(la, sched.Options{
				Platform:         commonOpts.Platform,
				WaitCompletion:   opts.waitCompletion,
				RTEConfigData:    commonOpts.RTEConfigData,
				PullIfNotPresent: commonOpts.PullIfNotPresent,
			})
			if err != nil {
				// intentionally keep going to remove as much as possible
				log.Printf("error removing: %v", err)
			}
			err = rte.Remove(la, rte.Options{
				Platform:         commonOpts.Platform,
				WaitCompletion:   opts.waitCompletion,
				RTEConfigData:    commonOpts.RTEConfigData,
				PullIfNotPresent: commonOpts.PullIfNotPresent,
			})
			if err != nil {
				// intentionally keep going to remove as much as possible
				log.Printf("error removing: %v", err)
			}
			err = api.Remove(la, api.Options{
				Platform: commonOpts.Platform,
			})
			if err != nil {
				// intentionally keep going to remove as much as possible
				log.Printf("error removing: %v", err)
			}
			return nil
		},
		Args: cobra.NoArgs,
	}
	remove.PersistentFlags().BoolVarP(&opts.waitCompletion, "wait", "W", false, "wait for removal to be all completed.")
	remove.AddCommand(NewRemoveAPICommand(commonOpts, opts))
	remove.AddCommand(NewRemoveSchedulerPluginCommand(commonOpts, opts))
	remove.AddCommand(NewRemoveTopologyUpdaterCommand(commonOpts, opts))
	return remove
}

func NewDeployAPICommand(commonOpts *CommonOptions, opts *deployOptions) *cobra.Command {
	deploy := &cobra.Command{
		Use:   "api",
		Short: "deploy the APIs needed for topology-aware-scheduling",
		RunE: func(cmd *cobra.Command, args []string) error {
			la := logAdapter{
				log:      commonOpts.Log,
				debugLog: commonOpts.DebugLog,
			}
			if err := api.Deploy(la, api.Options{Platform: commonOpts.Platform}); err != nil {
				return err
			}
			return nil
		},
		Args: cobra.NoArgs,
	}
	return deploy
}

func NewDeploySchedulerPluginCommand(commonOpts *CommonOptions, opts *deployOptions) *cobra.Command {
	deploy := &cobra.Command{
		Use:   "scheduler-plugin",
		Short: "deploy the scheduler plugin needed for topology-aware-scheduling",
		RunE: func(cmd *cobra.Command, args []string) error {
			la := logAdapter{
				log:      commonOpts.Log,
				debugLog: commonOpts.DebugLog,
			}
			return sched.Deploy(la, sched.Options{
				Platform:         commonOpts.Platform,
				WaitCompletion:   opts.waitCompletion,
				RTEConfigData:    commonOpts.RTEConfigData,
				PullIfNotPresent: commonOpts.PullIfNotPresent,
			})
		},
		Args: cobra.NoArgs,
	}
	return deploy
}

func NewDeployTopologyUpdaterCommand(commonOpts *CommonOptions, opts *deployOptions) *cobra.Command {
	deploy := &cobra.Command{
		Use:   "topology-updater",
		Short: "deploy the topology updater needed for topology-aware-scheduling",
		RunE: func(cmd *cobra.Command, args []string) error {
			la := logAdapter{
				log:      commonOpts.Log,
				debugLog: commonOpts.DebugLog,
			}
			return rte.Deploy(la, rte.Options{
				Platform:         commonOpts.Platform,
				WaitCompletion:   opts.waitCompletion,
				RTEConfigData:    commonOpts.RTEConfigData,
				PullIfNotPresent: commonOpts.PullIfNotPresent,
			})
		},
		Args: cobra.NoArgs,
	}
	return deploy
}

func NewRemoveAPICommand(commonOpts *CommonOptions, opts *deployOptions) *cobra.Command {
	remove := &cobra.Command{
		Use:   "api",
		Short: "remove the APIs needed for topology-aware-scheduling",
		RunE: func(cmd *cobra.Command, args []string) error {
			la := logAdapter{
				log:      commonOpts.Log,
				debugLog: commonOpts.DebugLog,
			}
			if err := api.Remove(la, api.Options{Platform: commonOpts.Platform}); err != nil {
				return err
			}
			return nil
		},
		Args: cobra.NoArgs,
	}
	return remove
}

func NewRemoveSchedulerPluginCommand(commonOpts *CommonOptions, opts *deployOptions) *cobra.Command {
	remove := &cobra.Command{
		Use:   "scheduler-plugin",
		Short: "remove the scheduler plugin needed for topology-aware-scheduling",
		RunE: func(cmd *cobra.Command, args []string) error {
			la := logAdapter{
				log:      commonOpts.Log,
				debugLog: commonOpts.DebugLog,
			}
			return sched.Remove(la, sched.Options{
				Platform:         commonOpts.Platform,
				WaitCompletion:   opts.waitCompletion,
				RTEConfigData:    commonOpts.RTEConfigData,
				PullIfNotPresent: commonOpts.PullIfNotPresent,
			})
		},
		Args: cobra.NoArgs,
	}
	return remove
}

func NewRemoveTopologyUpdaterCommand(commonOpts *CommonOptions, opts *deployOptions) *cobra.Command {
	remove := &cobra.Command{
		Use:   "topology-updater",
		Short: "remove the topology updater needed for topology-aware-scheduling",
		RunE: func(cmd *cobra.Command, args []string) error {
			la := logAdapter{
				log:      commonOpts.Log,
				debugLog: commonOpts.DebugLog,
			}
			return rte.Remove(la, rte.Options{
				Platform:         commonOpts.Platform,
				WaitCompletion:   opts.waitCompletion,
				RTEConfigData:    commonOpts.RTEConfigData,
				PullIfNotPresent: commonOpts.PullIfNotPresent,
			})
		},
		Args: cobra.NoArgs,
	}
	return remove
}

func deployOnCluster(commonOpts *CommonOptions, opts *deployOptions) error {
	la := logAdapter{
		log:      commonOpts.Log,
		debugLog: commonOpts.DebugLog,
	}
	if err := api.Deploy(la, api.Options{
		Platform: commonOpts.Platform,
	}); err != nil {
		return err
	}
	if err := rte.Deploy(la, rte.Options{
		Platform:         commonOpts.Platform,
		WaitCompletion:   opts.waitCompletion,
		RTEConfigData:    commonOpts.RTEConfigData,
		PullIfNotPresent: commonOpts.PullIfNotPresent,
	}); err != nil {
		return err
	}
	if err := sched.Deploy(la, sched.Options{
		Platform:         commonOpts.Platform,
		WaitCompletion:   opts.waitCompletion,
		RTEConfigData:    commonOpts.RTEConfigData,
		PullIfNotPresent: commonOpts.PullIfNotPresent,
	}); err != nil {
		return err
	}
	return nil
}
