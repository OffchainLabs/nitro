import {ConfigComponent} from './Config';
import {ActionForm} from './ActionForm';
import {EventsComponent} from './Events';

function App() {
  return (
    <div className="h-screen flex bg-gradient-to-tr from-fuchsia-300 to-sky-500">
      <div className="w-1/4">
        <ActionForm/>
        <EventsComponent/>
      </div>
      <div className="w-3/4">
        <ConfigComponent/>
      </div>
    </div>
  )
}

export default App
