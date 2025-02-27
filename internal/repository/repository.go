package repository

type Repository interface {
	User() UserRepository
	Project() ProjectRepository
	Release() ReleaseRepository
	Environment() EnvironmentRepository
	Session() SessionRepository
	GitRepo() GitRepoRepository
	Cluster() ClusterRepository
	Database() DatabaseRepository
	HelmRepo() HelmRepoRepository
	Registry() RegistryRepository
	Infra() InfraRepository
	GitActionConfig() GitActionConfigRepository
	Invite() InviteRepository
	AuthCode() AuthCodeRepository
	DNSRecord() DNSRecordRepository
	PWResetToken() PWResetTokenRepository
	KubeIntegration() KubeIntegrationRepository
	BasicIntegration() BasicIntegrationRepository
	OIDCIntegration() OIDCIntegrationRepository
	OAuthIntegration() OAuthIntegrationRepository
	GCPIntegration() GCPIntegrationRepository
	AWSIntegration() AWSIntegrationRepository
	GithubAppInstallation() GithubAppInstallationRepository
	GithubAppOAuthIntegration() GithubAppOAuthIntegrationRepository
	SlackIntegration() SlackIntegrationRepository
	NotificationConfig() NotificationConfigRepository
	JobNotificationConfig() JobNotificationConfigRepository
	BuildEvent() BuildEventRepository
	KubeEvent() KubeEventRepository
	ProjectUsage() ProjectUsageRepository
	Onboarding() ProjectOnboardingRepository
	CredentialsExchangeToken() CredentialsExchangeTokenRepository
	BuildConfig() BuildConfigRepository
	Allowlist() AllowlistRepository
}
