// JSON responses from backend/internal/models/response.go.
export type AICredential = {
  id: string
  name: string
  provider: string
  base_url: string
  created_at: string
}

export type AIConfig = {
  id: string
  credential_id: string
  config_name: string
  credential_name: string
  model: string
  temperature: number
  created_at: string
}
