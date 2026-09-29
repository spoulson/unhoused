import { createBrowserRouter } from 'react-router-dom'
import { Layout } from './Layout'
import { HomePage } from './pages/HomePage'
import { JobStatusPage } from './pages/JobStatusPage'
import { ProfilePage } from './pages/ProfilePage'

export const router = createBrowserRouter([
  {
    path: '/',
    element: <Layout />,
    children: [
      { index: true, element: <HomePage /> },
      { path: 'profile/:profileSlug', element: <ProfilePage /> },
      { path: 'profile/:profileSlug/jobs/:jobId', element: <JobStatusPage /> },
    ],
  },
])
