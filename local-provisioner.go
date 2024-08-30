package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/sig-storage-lib-external-provisioner/v7/controller"
)

type LocalProvisioner struct {
    Client kubernetes.Interface
}

func (p *LocalProvisioner) Provision(ctx context.Context, options controller.ProvisionOptions) (*corev1.PersistentVolume, controller.ProvisioningState, error) {
    path, ok := options.StorageClass.Parameters["path"]
    if !ok {
        return nil, controller.ProvisioningFinished, errors.New("Parameter 'path' not found")
    }

    // Generate a name for the PV and corresponding directory
    pvName := fmt.Sprintf("%s-%s", options.PVC.Name, uuid.NewUUID())
    dirPath := fmt.Sprintf("%s/%s", path, pvName)

    // Create a new PV based on the PVC's request
    pv := &corev1.PersistentVolume{
        ObjectMeta: metav1.ObjectMeta{
            Name: pvName,
        },
        Spec: corev1.PersistentVolumeSpec{
            Capacity: corev1.ResourceList{
                corev1.ResourceStorage: options.PVC.Spec.Resources.Requests[corev1.ResourceStorage],
            },
            AccessModes:                   options.PVC.Spec.AccessModes,
            PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
            StorageClassName:              options.StorageClass.Name,
            VolumeMode:                    options.PVC.Spec.VolumeMode,
            PersistentVolumeSource: corev1.PersistentVolumeSource{
                Local: &corev1.LocalVolumeSource{
                    Path: dirPath,
                },
            },
            NodeAffinity: &corev1.VolumeNodeAffinity{
                Required: &corev1.NodeSelector{
                    NodeSelectorTerms: []corev1.NodeSelectorTerm{
                        {
                            MatchExpressions: []corev1.NodeSelectorRequirement{
                                {
                                    Key:      corev1.LabelHostname,
                                    Operator: corev1.NodeSelectorOpIn,
                                    Values:   []string{os.Getenv("NODE_NAME")},
                                },
                            },
                        },
                    },
                },
            },
        },
    }

    // Create the directory with the appropriate permissions
    fmt.Printf("Creating directory: %s\n", dirPath)
    err := os.Mkdir(dirPath, 0775)
    if err != nil {
        return pv, controller.ProvisioningFinished, err
    }
    fmt.Printf("Directory created successfully: %s\n", dirPath)

    return pv, controller.ProvisioningFinished, nil
}

func (p *LocalProvisioner) Delete(ctx context.Context, volume *corev1.PersistentVolume) error {
    fmt.Printf("Deleting directory: %s\n", volume.Spec.Local.Path)
    err := os.RemoveAll(volume.Spec.Local.Path)
    if err != nil {
        return err
    }
    fmt.Printf("Directory deleted successfully: %s\n", volume.Spec.Local.Path)
    return nil
}

func main() {
    // Get the Kubernetes client configuration
    config, err := rest.InClusterConfig()
    if err != nil {
        panic(err.Error())
    }

    clientset := kubernetes.NewForConfigOrDie(config)

    localProvisioner := &LocalProvisioner{Client: clientset}

    // Create a new provisioner controller
    provisionController := controller.NewProvisionController(
        clientset,
        "local-provisioner",
        localProvisioner,
        controller.LeaderElection(false),
    )

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Setup signal handling to gracefully stop the provisioner
    stopCh := make(chan os.Signal, 1)
    signal.Notify(stopCh, os.Interrupt)
    go func() {
        <-stopCh
        cancel()
    }()

    provisionController.Run(ctx)
}
