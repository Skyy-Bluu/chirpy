module github.com/Skyy-Bluu/chirpy

go 1.25.4

replace github.com/Skyy-Bluu/chirpy/internal/database v0.0.0 => ./internal/database
replace github.com/Skyy-Bluu/chirpy/internal/auth v0.0.0 => ./internal/auth

require github.com/Skyy-Bluu/chirpy/internal/database v0.0.0
require github.com/Skyy-Bluu/chirpy/internal/auth v0.0.0

require (
	github.com/alexedwards/argon2id v1.0.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/lib/pq v1.11.2 // indirect
	golang.org/x/crypto v0.14.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
)
