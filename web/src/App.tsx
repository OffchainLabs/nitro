import {ConfigComponent} from './Config';
import {ActionForm} from './ActionForm';
import {EventsComponent} from './Events';
import {SidebarConfig} from './SidebarConfig';
import { AssertionsChain } from './AssertionChain';

function App() {
  return (
    <div className="h-screen flex bg-gradient-to-tr from-fuchsia-300 to-sky-500">
      <div className="w-1/4">
        <SidebarConfig/>
        <ActionForm/>
        <EventsComponent/>
      </div>
      <div className="w-3/4">
        <AssertionsChain/>
      </div>
    </div>
  )
}

export default App
