package get_service_values

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/flant/werf/cmd/werf/common"
	"github.com/flant/werf/cmd/werf/common/docker_authorizer"
	"github.com/flant/werf/pkg/deploy"
	"github.com/flant/werf/pkg/docker"
	"github.com/flant/werf/pkg/git_repo"
	"github.com/flant/werf/pkg/lock"
	"github.com/flant/werf/pkg/logger"
	"github.com/flant/werf/pkg/project_tmp_dir"
	"github.com/flant/werf/pkg/ssh_agent"
	"github.com/flant/werf/pkg/true_git"
	"github.com/flant/werf/pkg/util"
	"github.com/flant/werf/pkg/werf"
	"github.com/spf13/cobra"
)

var CmdData struct {
	RegistryUsername string
	RegistryPassword string
}

var CommonCmdData common.CmdData

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-service-values",
		Short: "Get service values yaml generated by werf for helm chart during deploy",
		Long: common.GetLongCommandDescription(`Get service values generated by werf for helm chart during deploy.

These values includes project name, docker images ids and other.`),
		DisableFlagsInUseLine: true,
		Annotations: map[string]string{
			common.CmdEnvAnno: common.EnvsDescription(common.WerfSecretKey, common.WerfDockerConfig, common.WerfHome, common.WerfTmp),
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runGetServiceValues(); err != nil {
				return fmt.Errorf("get-service-values failed: %s", err)
			}

			return nil
		},
	}

	common.SetupDir(&CommonCmdData, cmd)
	common.SetupTmpDir(&CommonCmdData, cmd)
	common.SetupHomeDir(&CommonCmdData, cmd)
	common.SetupSSHKey(&CommonCmdData, cmd)

	cmd.Flags().StringVarP(&CmdData.RegistryUsername, "registry-username", "", "", "Docker registry username")
	cmd.Flags().StringVarP(&CmdData.RegistryPassword, "registry-password", "", "", "Docker registry password")

	common.SetupTag(&CommonCmdData, cmd)
	common.SetupEnvironment(&CommonCmdData, cmd)
	common.SetupNamespace(&CommonCmdData, cmd)
	common.SetupImagesRepo(&CommonCmdData, cmd)

	return cmd
}

func runGetServiceValues() error {
	logger.Out = ioutil.Discard

	if err := werf.Init(*CommonCmdData.TmpDir, *CommonCmdData.HomeDir); err != nil {
		return fmt.Errorf("initialization error: %s", err)
	}

	if err := lock.Init(); err != nil {
		return err
	}

	if err := deploy.Init(); err != nil {
		return err
	}

	if err := true_git.Init(); err != nil {
		return err
	}

	if err := ssh_agent.Init(*CommonCmdData.SSHKeys); err != nil {
		return fmt.Errorf("cannot initialize ssh-agent: %s", err)
	}

	if err := docker.Init(docker_authorizer.GetHomeDockerConfigDir()); err != nil {
		return err
	}

	projectDir, err := common.GetProjectDir(&CommonCmdData)
	if err != nil {
		return fmt.Errorf("getting project dir failed: %s", err)
	}

	werfConfig, err := common.GetWerfConfig(projectDir)
	if err != nil {
		return fmt.Errorf("cannot parse werf config: %s", err)
	}

	imagesRepo := common.GetOptionalImagesRepo(werfConfig.Meta.Project, &CommonCmdData)
	withoutRegistry := true

	if imagesRepo != "" {
		withoutRegistry = false

		var err error

		projectTmpDir, err := project_tmp_dir.Get()
		if err != nil {
			return fmt.Errorf("getting project tmp dir failed: %s", err)
		}
		defer project_tmp_dir.Release(projectTmpDir)

		dockerAuthorizer, err := docker_authorizer.GetCommonDockerAuthorizer(projectTmpDir, CmdData.RegistryUsername, CmdData.RegistryPassword)
		if err != nil {
			return err
		}

		if err := dockerAuthorizer.Login(imagesRepo); err != nil {
			return fmt.Errorf("docker login failed: %s", err)
		}
	}

	if imagesRepo == "" {
		imagesRepo = "IMAGES_REPO"
	}

	environment := *CommonCmdData.Environment
	if environment == "" {
		environment = "ENV"
	}

	namespace, err := common.GetKubernetesNamespace(*CommonCmdData.Namespace, environment, werfConfig)
	if err != nil {
		return err
	}

	tag, err := common.GetDeployTag(&CommonCmdData)
	if err != nil {
		return err
	}

	localGit := &git_repo.Local{Path: projectDir, GitDir: filepath.Join(projectDir, ".git")}

	images := deploy.GetImagesInfoGetters(werfConfig.Images, imagesRepo, tag, withoutRegistry)

	serviceValues, err := deploy.GetServiceValues(werfConfig.Meta.Project, imagesRepo, namespace, tag, localGit, images, deploy.ServiceValuesOptions{})
	if err != nil {
		return fmt.Errorf("error creating service values: %s", err)
	}

	fmt.Printf("%s", util.DumpYaml(serviceValues))

	return nil
}
