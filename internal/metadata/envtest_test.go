package metadata

import (
	"log/slog"
	"os"
	"testing"

	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	client *kubernetes.Clientset
)

func TestMain(m *testing.M) {
	dir, err := envtest.SetupEnvtestDefaultBinaryAssetsDirectory()
	if err != nil {
		panic(err)
	}

	env := envtest.Environment{
		DownloadBinaryAssets:  true,
		BinaryAssetsDirectory: dir,
	}
	cfg, err := env.Start()
	if err != nil {
		panic(err)
	}
	defer env.Stop()

	client = kubernetes.NewForConfigOrDie(cfg)
	v, err := client.DiscoveryClient.ServerVersion()
	if err != nil {
		panic(err)
	}
	slog.Info("testing with kubernetes", "version", v)

	code := m.Run()

	if err = env.Stop(); err != nil {
		panic(err)
	}

	os.Exit(code)
}
