import {ConfigComponent} from './Config';
import {ActionForm} from './ActionForm';
import {SidebarConfig} from './SidebarConfig';
import {Visualization} from './Visualization';
import {EventsList} from './EventsList';

function App() {
  return (
    <div className="h-screen flex bg-white">
      <div className="w-1/4 p-4">
        <SidebarConfig/>
        <ActionForm/>
        <EventsList/>
      </div>
      <div className="w-3/4">
        <Visualization/>
      </div>
    </div>
  )
}

export default App
