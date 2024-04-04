package cmd

import (
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
	Run: func(_ *cobra.Command, _ []string) {
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

		slog.Info("Listening", "port", strconv.Itoa(viper.GetInt("port")), "path", viper.GetString("path"))
		http.Handle(viper.GetString("path"), promhttp.Handler())
		if err := http.ListenAndServe(":"+strconv.Itoa(viper.GetInt("port")), nil); err != nil {
			slog.Warn("Prometheus webserver ended not normally", "error", err)
		}

		slog.Info("Finished process")
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
	rootCmd.Flags().String("path", "/metrics", "Path to expose metrics on")
	rootCmd.Flags().Int("port", 8080, "Port to listen on")
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
