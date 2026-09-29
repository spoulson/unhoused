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
      { path: 'profiles/:profileSlug', element: <ProfilePage /> },
      { path: 'profiles/:profileSlug/jobs/:jobId', element: <JobStatusPage /> },
    ],
  },
])
