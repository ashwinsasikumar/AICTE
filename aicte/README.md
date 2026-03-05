# React Frontend - PDF Link Validator

Modern React application for bulk PDF link validation with a clean, responsive UI.

## Features

- ✅ Bulk URL input (newline or comma-separated)
- ✅ Real-time validation results
- ✅ Loading indicators & progress tracking
- ✅ Summary statistics (total, valid, invalid)
- ✅ Export results as CSV
- ✅ Copy invalid links to clipboard
- ✅ Responsive design with Tailwind CSS
- ✅ Error handling for network issues

## Project Structure

```
aicte/
├── public/
│   ├── index.html
│   └── manifest.json
├── src/
│   ├── App.js            # Main application component
│   ├── App.css           # Component styles
│   ├── index.js          # Entry point
│   └── index.css         # Global styles + Tailwind
├── Dockerfile            # Production Docker config
├── nginx.conf            # Nginx configuration
└── package.json          # Dependencies
```

## Available Scripts

In the project directory, you can run:

### `npm start`

Runs the app in the development mode.\
Open [http://localhost:3000](http://localhost:3000) to view it in your browser.

The page will reload when you make changes.\
You may also see any lint errors in the console.

### `npm test`

Launches the test runner in the interactive watch mode.\
See the section about [running tests](https://facebook.github.io/create-react-app/docs/running-tests) for more information.

### `npm run build`

Builds the app for production to the `build` folder.\
It correctly bundles React in production mode and optimizes the build for the best performance.

The build is minified and the filenames include the hashes.\
Your app is ready to be deployed!

## Building for Production

```bash
# Create optimized production build
npm run build

# The build folder is ready to be deployed
```

## Environment Configuration

Create a `.env` file in the root directory:

```env
REACT_APP_API_URL=http://localhost:8080
```

Then update App.js to use it:
```javascript
const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';
```

## Docker Deployment

```bash
# Build image
docker build -t pdf-validator-frontend .

# Run container
docker run -p 3000:80 pdf-validator-frontend
```

The app will be served by Nginx on port 80 (mapped to port 3000).

## Dependencies

- **react**: ^19.2.4
- **react-dom**: ^19.2.4
- **tailwindcss**: ^3.3.5
- **react-scripts**: 5.0.1

## Usage

1. Paste URLs into the textarea (one per line or comma-separated)
2. Click "Validate Links"
3. View results in the table
4. Export to CSV or copy invalid links

## Styling

Uses Tailwind CSS for utility-first styling. Configuration in:
- `tailwind.config.js`
- `postcss.config.js`

## Browser Support

- Chrome (latest)
- Firefox (latest)
- Safari (latest)
- Edge (latest)

## Learn More

You can learn more in the [Create React App documentation](https://facebook.github.io/create-react-app/docs/getting-started).

To learn React, check out the [React documentation](https://reactjs.org/).
