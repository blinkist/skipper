package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	dockerpty "github.com/blinkist/go-dockerpty"
	docker "github.com/fsouza/go-dockerclient"
	"golang.org/x/crypto/ssh"

	"github.com/blinkist/skipper/aws/ec2client"
	"github.com/blinkist/skipper/aws/ec2resource"
	"github.com/blinkist/skipper/aws/ecsclient"
	"github.com/blinkist/skipper/helpers"
)

const (
	relConfigSSHPath = ".ssh"

	sshTimeout = 10 * time.Second
)

var (
	logger *log.Logger

	// DEBUGCLUSTERNAME is the name of the ECS Cluster to copy tasks to. TODO: this should be more configurable
	DEBUGCLUSTERNAME = "DEBUG"
)

func init() {
	logger = log.New(os.Stderr, " - ", log.LstdFlags)
}

// GetKeypairName provides the name of the keypair on AWS
// Todo: make this more configurable ?
func GetKeypairName() *string {
	identifier := fmt.Sprintf("skipper-%s", os.Getenv("USER"))
	return &identifier
}

// GetIdentifier returns an identifier used for the copied task where the user's name is included
func GetIdentifier(taskdef *string) *string {
	identifier := fmt.Sprintf("skipper-%s-%s", os.Getenv("USER"), *taskdef)
	return &identifier
}

// ShellSelectTask is an interactive method which asks the user to select one of the few shell tasks
// Todo: Create a distinct selection of tasks by task version
func ShellSelectTask() (*ecsclient.TaskInfo, error) {
	ecs := ecsclient.GetInstance()
	cluster, service := helpers.ServicePicker(ecs, nil)

	var taskinfos []*ecsclient.TaskInfo
	var err error

	taskinfos, err = ecs.GetContainerInstances(&cluster, &service)
	if err != nil {
		return nil, err
	}

	var selectString []string
	for _, ti := range taskinfos {
		mystr := fmt.Sprintf("%s:%s\t - %s - %s", *ti.IpAddress, strconv.FormatInt(*ti.Hostport0, 10), *ti.TaskDefinitionArn, *ti.Ec2InstanceId)
		selectString = append(selectString, mystr)
	}

	choice, _ := strconv.Atoi(helpers.PickOption(selectString, "Please choose a task to run"))

	return taskinfos[choice], nil
}

// InvokeShell method called to start invoking a shell inside a newly created docker
func InvokeShell() error {
	livetask, err := ShellSelectTask()
	if err != nil {
		return err
	}

	tasks := GetRunningTasks(livetask.TaskDefinitionArn)

	if len(tasks) > 0 {
		InvokeShellOnActiveTask(tasks)
	} else {

		instancecopy := StartInstance(livetask)

		taskcopy := StartTaskOnInstance(livetask, instancecopy)

		DockerStart(instancecopy, taskcopy)
	}
	return nil
}

// InvokeShellOnActiveTask invokes a shell on an already running debug task
func InvokeShellOnActiveTask(tasks []*ecs.Task) {
	ecs := ecsclient.GetInstance()
	ec2cl := ec2client.GetInstance()

	clusterParts := strings.Split(*tasks[0].ClusterArn, "/")
	clusterName := clusterParts[len(clusterParts)-1]

	instanceId, err := ecs.GetInstanceIDForContainerArn(&clusterName, tasks[0].ContainerInstanceArn)

	if err != nil {
		logger.Println(err)
		os.Exit(1)
	}
	instance, err := ec2cl.DescribeInstance(instanceId)

	if err != nil {
		logger.Println(err)
		os.Exit(1)
	}

	DockerStart(instance, tasks[0])

}

// GetRunningTasks gets running DEBUG tasks belonging to the user executing skipper
func GetRunningTasks(taskdefinition *string) []*ecs.Task {
	ecs := ecsclient.GetInstance()
	tasks, err := ecs.GetClusterTasksWithDefinition(&DEBUGCLUSTERNAME, taskdefinition)
	if err != nil {
		logger.Println("Could not retrieve tasks")
		logger.Println(err)
		os.Exit(1)
	}

	identifier := GetKeypairName()

	j := 0
	for i := range tasks {
		if tasks[i].StartedBy != nil {
			if *tasks[i].StartedBy == *identifier {
				tasks[j] = tasks[i]
				j++
			}
		}
	}
	tasks = tasks[:j]

	return tasks
}

// StartInstance starts an EC2 Instance with keypair belonging to a user
func StartInstance(livetask *ecsclient.TaskInfo) *ec2.Instance {
	identifier := GetIdentifier(livetask.TaskDefinitionArn)

	userdata := fmt.Sprintf(`#!/bin/bash
echo ECS_CLUSTER=%s >> /etc/ecs/ecs.config
echo ECS_INSTANCE_ATTRIBUTES={\"group\": \"%s\"} >> /etc/ecs/ecs.config
`, DEBUGCLUSTERNAME, *identifier)

	ec2cl := ec2client.GetInstance()

	// SelectTask(ecs2client_)
	keypairname := GetKeypairName()
	key_on_aws := ec2cl.KeypairExists(keypairname)
	key_on_fs := PrivateKeyExists(*keypairname)

	if !(key_on_aws && key_on_fs) {
		logger.Println("We do not have a valid keypair")
		os.Exit(1)
	}

	ec2resource := ec2resource.New(livetask.Ec2InstanceId, ec2cl)
	ec2resource.RefreshInstance()

	input := ec2client.StartInstanceInput{
		IamInstanceProfileArn: ec2resource.GetIamInstanceProfile(),
		ImageID:               ec2resource.GetImageID(),
		InstanceType:          aws.String("t2.large"),
		KeyName:               keypairname,
		SecurityGroupIds:      ec2resource.GetSecurityGroupIDS(),
		SubnetID:              ec2resource.GetSubnetID(),
		UserData:              &userdata,
		TagValue:              identifier,
	}

	ec2instance, err := ec2cl.StartInstance(&input)

	if err != nil {
		logger.Printf("Error happened %s\n", err)
		os.Exit(1)
	}

	return ec2instance
}

// StartTaskOnInstance starts task on ec2 instance ( debug instance )
func StartTaskOnInstance(livetask *ecsclient.TaskInfo, ec2instance *ec2.Instance) *ecs.Task {
	ecsclient_ := ecsclient.GetInstance()

	varInstances := make([]*ec2.Instance, 1)
	varInstances[0] = ec2instance

	var containerinstances []*ecs.ContainerInstance
	var err error
	for i := 0; i < 30; i++ {
		containerinstances, err = ecsclient_.DescribeContainerInstances(&DEBUGCLUSTERNAME, varInstances)
		if len(containerinstances) == 1 {
			break
		}
		logger.Println("Waiting for the instance to join the ECS Cluster")
		time.Sleep(10 * time.Second)

	}

	if err != nil {
		logger.Printf("Error happened DescribeContainerInstances %s\n", err)
		os.Exit(1)
	}
	if len(containerinstances) != 1 {
		logger.Printf("Problem  len containerinstances %s\n", containerinstances)
		os.Exit(1)
	}

	taskrolearn, _ := ecsclient_.GetTaskRoleArn(livetask.TaskDefinitionArn)

	sto, err2 := ecsclient_.StartTaskOnContainerInstance(&DEBUGCLUSTERNAME, livetask.TaskDefinitionArn, containerinstances[0].ContainerInstanceArn, taskrolearn, ec2instance.KeyName)

	if len(sto.Failures) > 0 || err2 != nil {
		logger.Println("Problem running task")
		logger.Println("Implement desctruction of EC2 Instance")
		os.Exit(1)
	}

	if len(sto.Tasks) != 1 {
		fmt.Println("We don't have one task running")
		os.Exit(1)
	}

	errwaitfortask := ecsclient_.WaitForTaskRunning(&DEBUGCLUSTERNAME, sto.Tasks[0].TaskArn)
	if errwaitfortask != nil {
		logger.Println("Task takes too long to start")
		os.Exit(1)
	}
	return sto.Tasks[0]
}

// GetUnixSocketPath returns newly created temporary socket for skipper's docker implemtation to listen to
func GetUnixSocketPath() (*string, *string) {
	dir, err := ioutil.TempDir("/tmp", "skipper")

	if err != nil {
		logger.Fatalf("unable to read Temporary Directory: %v", err)
	}
	localSocketPath := fmt.Sprintf("%s/skipper", dir)
	localSocket := fmt.Sprintf("unix://%s", localSocketPath)

	return &localSocket, &localSocketPath
}

// DockerStart takes care of creating an SSH Tunnel and forwarding the docket socket to be able to exec into the docker of the remote task
func DockerStart(ec2instance *ec2.Instance, task *ecs.Task) error {

	dockerClient, err := StartDockerTunnel(*ec2instance.PrivateIpAddress)
	if err != nil {
		panic(err)
	}

	//	conts, err := dockerClient.ListContainers(docker.ListContainersOptions{All: false})
	conts, err := dockerClient.ListContainers(docker.ListContainersOptions{All: false})

	if err != nil {
		panic(err)
	}

	for _, container := range conts {
		value, ok := container.Labels["com.amazonaws.ecs.task-arn"]
		if !ok {
			continue
		}
		if value == *task.TaskArn {
			exec, err := dockerClient.CreateExec(docker.CreateExecOptions{
				Container:    container.ID,
				AttachStdin:  true,
				AttachStdout: true,
				AttachStderr: true,
				Tty:          true,
				Cmd:          []string{"/bin/sh"},
			})

			if err != nil {
				logger.Println(err)
				os.Exit(1)
			}

			// Fire up the console
			if err = dockerpty.StartExec(dockerClient, exec); err != nil {
				// This is where we get stuck exiting the shell
				logger.Println("error execing container:", err)
			}
		}
		break
	}
	StopInstance(ec2instance)
	return nil
}

// StopInstance is an interactive method taking care of stopping a started debug instance
func StopInstance(ec2instance *ec2.Instance) {
	choicelist := []string{"Yes", "No"}
	choice := helpers.PickOption(choicelist, "Do you want the instance to terminate?")
	if choice == "Yes" {
		log.Printf("Terminating instance %s", *ec2instance.InstanceId)
		err := safeTerminateInstance(ec2instance)
		if err != nil {
			log.Fatalf("Could not terminate instance %s error: %v\n", *ec2instance.InstanceId, err)
		}
		log.Println("Succesfully deleted instance")
	}
}

// safeTerminateInstance is a private method which wraps around the ec2client terminate
// to make sure the instance is started by the executing skipper user
func safeTerminateInstance(ec2instance *ec2.Instance) error {
	userkeyname := GetKeypairName()
	if *ec2instance.KeyName != *userkeyname {
		logger.Fatalf("user not eligible to destroy instance; different keypair")
	}

	return ec2client.GetInstance().TerminateInstance(ec2instance)
}

// SetKeypair sets the keypair on both local filesystem and EC2
func SetKeypair() {
	EnsureSSHDir()

	ec2cl := ec2client.GetInstance()
	keyname := GetKeypairName()

	key_on_aws := ec2cl.KeypairExists(keyname)
	key_on_fs := PrivateKeyExists(*keyname)

	if key_on_aws && !key_on_fs {
		logger.Println("The keypair is not available locally, please run purge keypair")
		os.Exit(1)
	}

	if !key_on_aws && key_on_fs {
		logger.Println("The keypair is not available on AWS, just locally, please run purge keypair")
		os.Exit(1)
	}

	kpout, err := ec2cl.CreateKeypair(keyname)
	if err != nil {
		logger.Printf("Error creating keypair %s\n", err)

		os.Exit(1)
	} else {
		SetPrivateKey(*keyname, *kpout.KeyMaterial)
		logger.Printf("Succesfully set keypair %s \n", *keyname)
	}
}

// getSSHConfigDir gets the skipper ssh dir
func getSSHConfigDir() string {
	home, err := helpers.UnixHome()
	if err != nil {
		fmt.Println("Cannot determine homedir")
		os.Exit(1)
	}
	return filepath.Join(home, helpers.RelConfigPath, relConfigSSHPath)
}

// GetPrivateKeyPathForName gets the private ssh key for Name X
func GetPrivateKeyPathForName(name string) string {
	sshdir := getSSHConfigDir()
	return filepath.Join(sshdir, name)
}

// SetPrivateKey bools if private key for name {name} exists
func PrivateKeyExists(name string) bool {
	path := GetPrivateKeyPathForName(name)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

// SetPrivateKey sets the private key on local FS for {name}
func SetPrivateKey(name string, content string) error {
	EnsureSSHDir()
	path := GetPrivateKeyPathForName(name)
	d1 := []byte(content)
	err := ioutil.WriteFile(path, d1, 0600)
	return err
}

// DeletePrivateKey deletes private key for {name}
func DeletePrivateKey(name string) error {
	path := GetPrivateKeyPathForName(name)
	err := os.Remove(path)
	return err
}

// EnsureSSHDir makes sure skippers .ssh dir exists
func EnsureSSHDir() {
	err := helpers.EnsureConfigDir()
	if err != nil {
		logger.Printf("Cannot ensure Configdir %s", err)
		os.Exit(1)
	}

	configDir := helpers.GetConfigDir()

	path := fmt.Sprintf("%s/%s", *configDir, relConfigSSHPath)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 0700)
	}
}

type sshTunnelDialer struct {
	host string
}

func (d *sshTunnelDialer) Dial(network, addr string) (net.Conn, error) {
	sshAddr := d.host + ":22"
	// Build SSH client configuration
	cfg, err := makeSSHConfig()
	if err != nil {
		logger.Fatalf("Error configuring SSH: %v", err)
	}
	// Establish connection with SSH server
	conn, err := ssh.Dial("tcp", sshAddr, cfg)
	if err != nil {
		logger.Fatalf("Error establishing SSH connection: %v", err)
	}
	remote, err := conn.Dial("unix", "/var/run/docker.sock")
	if err != nil {
		logger.Fatalf("Error connecting to Docker socket: %v", err)
	}
	return remote, err
}

func makeSSHConfig() (*ssh.ClientConfig, error) {
	keypath := GetPrivateKeyPathForName(*GetKeypairName())

	key, err := ioutil.ReadFile(keypath)
	if err != nil {
		logger.Fatalf("unable to read private key: %v", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		logger.Fatalf("unable to parse private key: %v", err)
	}

	config := &ssh.ClientConfig{
		User: "ec2-user",
		Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: sshTimeout,
	}
	return config, nil
}

func StartDockerTunnel(ip string) (*docker.Client, error) {
	dialer := &sshTunnelDialer{
		host: ip,
	}
	newClient, err := docker.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		return nil, fmt.Errorf("Can't initialise Docker: %v", err)
	}
	// TODO(jonboulle): go-dockerclient actually ignores the Dialer
	// embedded in this transport and just replaces it with whatever is set
	// on its own .Dialer. so this is somewhat redundant. but we still need
	// to trick it into feeding the Dialer through to the HTTPClient.
	newClient.Dialer = dialer
	newClient.WithTransport(func() *http.Transport {
		return &http.Transport{
			Dial: dialer.Dial,
		}
	})
	return newClient, nil
}
