import reactHotLoader, { hot } from 'react-hot-loader';
import preact from 'preact';

import Root from 'app/components/root';

reactHotLoader.preact(preact);

export default hot(module)(Root);
