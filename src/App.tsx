import React, { useEffect, useState } from 'react';
import { Settings, Home, User } from 'lucide-react';
import { useRegisterSW } from 'virtual:pwa-register/react';

function App() {
  const [isOnline, setIsOnline] = useState(navigator.onLine);
  const {
    needRefresh: [needRefresh],
    updateServiceWorker,
  } = useRegisterSW();

  useEffect(() => {
    const handleOnline = () => setIsOnline(true);
    const handleOffline = () => setIsOnline(false);

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, []);

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <div className="flex items-center">
              <Home className="h-6 w-6 text-gray-600" />
              <h1 className="ml-2 text-xl font-semibold text-gray-900">PWA App</h1>
            </div>
            <div className="flex items-center space-x-4">
              <User className="h-6 w-6 text-gray-600" />
              <Settings className="h-6 w-6 text-gray-600" />
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-lg font-medium text-gray-900">App Status</h2>
            <span 
              className={`px-3 py-1 rounded-full text-sm font-medium ${
                isOnline ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
              }`}
            >
              {isOnline ? 'Online' : 'Offline'}
            </span>
          </div>

          {needRefresh && (
            <div className="mt-4 bg-blue-50 p-4 rounded-md">
              <div className="flex">
                <div className="flex-shrink-0">
                  <Settings className="h-5 w-5 text-blue-400" />
                </div>
                <div className="ml-3">
                  <h3 className="text-sm font-medium text-blue-800">
                    Update available
                  </h3>
                  <div className="mt-2">
                    <button
                      className="bg-blue-500 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                      onClick={() => updateServiceWorker(true)}
                    >
                      Update now
                    </button>
                  </div>
                </div>
              </div>
            </div>
          )}

          <div className="mt-6">
            <p className="text-gray-600">
              This is a Progressive Web App built with React and Vite. It works offline and can be installed on your device.
            </p>
          </div>
        </div>
      </main>
    </div>
  );
}

export default App;