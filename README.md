# README

A simple expense capturing PWA, using Google Sheets as the backend.

## Environment Variables

| Name                     | Description |
|------------------------------|-------------|
| EXPENSER_OIDC_CALLBACK_URL   | The OIDC callback URL. |
| EXPENSER_OIDC_CLIENT_ID      | The OIDC client ID. |
| EXPENSER_USERFILE            | The path to the file containing authorized user emails. |
| EXPENSER_SHEET_ID            | The ID of the Google Sheet backend. |
| EXPENSER_AUTHNZ_DISABLED     | Disables authentication and authorization if set. |
| EXPENSER_NO_SHEETS_API       | Disables interaction with Google Sheets API if set. |
