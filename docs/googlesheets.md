# Google Sheets Integration Guide

The Cocktail Bot supports Google Sheets as a database backend, allowing you to store user data in a spreadsheet.

## Setup Instructions

### 1. Create a Google Cloud Project

1. Go to the [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the Google Sheets API for your project

### 2. Create Service Account Credentials

1. In your Google Cloud Project, go to "IAM & Admin" > "Service Accounts"
2. Click "Create Service Account"
3. Enter a name and description for your service account
4. Grant the service account the "Editor" role for your project
5. Click "Done" to create the service account
6. Click on the service account email to view its details
7. Go to the "Keys" tab and click "Add Key" > "Create new key"
8. Select JSON as the key type and click "Create"
9. The credentials file will be downloaded to your computer
10. Rename it to `credentials.json` and move it to your bot's directory

### 3. Create a Google Sheet

1. Go to [Google Sheets](https://sheets.google.com/) and create a new spreadsheet
2. Share the spreadsheet with the service account email (with Editor permissions)
3. Note the spreadsheet ID from the URL: `https://docs.google.com/spreadsheets/d/YOUR_SPREADSHEET_ID/edit`

### 4. Initialize the Sheet

Use the demo tool to set up the sheet:

```bash
go run cmd/demo-googlesheet/main.go -creds credentials.json -sheet-id YOUR_SPREADSHEET_ID -action setup
```

This will:
- Create a tab named "Sheet1" if it doesn't exist
- Add headers: ID, Email, Date Added, Redeemed
- Format the header row

### 5. Configure the Bot

Update your `config.yaml` to use Google Sheets:

```yaml
database:
  type: "googlesheet"
  connection_string: "credentials.json|YOUR_SPREADSHEET_ID|Sheet1"
```

## Using the Demo Tool

The `demo-googlesheet` tool allows you to interact with the Google Sheet directly:

```bash
# Show all users in the sheet
go run cmd/demo-googlesheet/main.go -creds credentials.json -sheet-id YOUR_SPREADSHEET_ID -action show

# Add a new user
go run cmd/demo-googlesheet/main.go -creds credentials.json -sheet-id YOUR_SPREADSHEET_ID -action add -email user@example.com

# Check if a user exists
go run cmd/demo-googlesheet/main.go -creds credentials.json -sheet-id YOUR_SPREADSHEET_ID -action check -email user@example.com

# Mark a user as having redeemed their cocktail
go run cmd/demo-googlesheet/main.go -creds credentials.json -sheet-id YOUR_SPREADSHEET_ID -action redeem -email user@example.com
```

## Sheet Structure

The Google Sheet has the following columns:

1. **ID**: A unique identifier for each user
2. **Email**: The user's email address (used for lookups)
3. **Date Added**: When the user was added to the sheet (RFC3339 format)
4. **Redeemed**: When the user redeemed their cocktail (RFC3339 format, empty if not redeemed)

## Troubleshooting

- **Permission Issues**: Make sure the service account has Editor access to the spreadsheet
- **API Errors**: Check that the Google Sheets API is enabled for your project
- **Invalid Credentials**: Ensure the credentials.json file is correctly formatted and has the necessary permissions
- **Rate Limiting**: Google Sheets API has usage limits; consider switching to a database for high-traffic scenarios

## Performance Considerations

Google Sheets is a convenient option for small-scale deployments, but it has limitations:

- API quotas restrict the number of requests per minute
- Performance will degrade with large numbers of users
- Concurrent writes can cause conflicts

For high-traffic bots, consider using another database backend like PostgreSQL or MongoDB.