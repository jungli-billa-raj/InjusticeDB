package db

type Repositories struct {
	Users         UserRepository
	Incidents     IncidentRepository
	Culprits      CulpritRepository
	Assets        AssetRepository
	Verifications VerificationRepository
	Messaging     MessagingRepository
	Comments      CommentRepository
	Targets       TargetRepository
}
