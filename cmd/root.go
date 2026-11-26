package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/akquinet/oxidized-exporter/oxidized"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "oxidized-exporter",
	Short: "Oxidized exporter for Prometheus",
	RunE: func(cmd *cobra.Command, _ []string) error {
		logLevel := slog.LevelWarn
		if viper.GetBool("debug") {
			logLevel = slog.LevelDebug
		} else if viper.GetBool("verbose") {
			logLevel = slog.LevelInfo
		}
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel, AddSource: true}))
		slog.SetDefault(logger)

		if viper.GetString("user") == "" || viper.GetString("pass") == "" {
			slog.Warn("No username or password given, using oxidized without basic authentication")
		}

		oxidizedClient := oxidized.NewOxidizedClient(viper.GetString("url"), viper.GetString("user"), viper.GetString("pass"))
		collector := oxidized.NewOxidizedCollector(oxidizedClient)
		prometheus.MustRegister(collector)

		listen := viper.GetString("listen")
		if viper.IsSet("port") {
			if viper.IsSet("listen") {
				return errors.New("--port and --listen are mutually exclusive, use --listen")
			}
			listen = ":" + strconv.Itoa(viper.GetInt("port"))
			if !cmd.Flags().Changed("port") {
				slog.Warn("OXIDIZED_EXPORTER_PORT is deprecated, use OXIDIZED_EXPORTER_LISTEN instead")
			}
		}

		slog.Info("Listening", "address", listen, "path", viper.GetString("path"))
		http.Handle(viper.GetString("path"), promhttp.Handler())
		if err := http.ListenAndServe(listen, nil); err != nil {
			return fmt.Errorf("prometheus webserver ended: %w", err)
		}

		slog.Info("Finished process")
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().String("listen", ":8080", "TCP address and port to listen on")
	rootCmd.Flags().Int("port", 8080, "Port to listen on (deprecated, use --listen)")
	if err := rootCmd.Flags().MarkDeprecated("port", "use --listen instead"); err != nil {
		panic(err)
	}
	rootCmd.Flags().String("path", "/metrics", "Path to expose metrics on")
	rootCmd.Flags().StringP("url", "U", "http://localhost:8888", "URL of oxidized API")
	rootCmd.Flags().StringP("user", "u", "", "Username for oxidized API")
	rootCmd.Flags().StringP("pass", "p", "", "Password for oxidized API")
	rootCmd.Flags().BoolP("debug", "d", false, "Enable debug logging")
	rootCmd.Flags().BoolP("verbose", "v", false, "Enable verbose logging")

	viper.AutomaticEnv()
	viper.SetEnvPrefix("OXIDIZED_EXPORTER")

	// bind all cobra flags to viper
	viper.BindPFlags(rootCmd.Flags())
}
