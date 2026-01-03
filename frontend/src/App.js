import React, { useState } from 'react';
import { Container, Header, Icon, Tab } from 'semantic-ui-react';
import 'semantic-ui-css/semantic.min.css';
import './App.css';
import Tab1 from './components/Tab1';
import Tab2 from './components/Tab2';
import Tab3 from "./components/Tab3";

function App() {
    const [activeTab, setActiveTab] = useState(0);

    const panes = [
        {
            menuItem: 'Вкладка 1',
            render: () => <Tab.Pane><Tab1 /></Tab.Pane>,
        },
        {
            menuItem: 'Вкладка 2',
            render: () => <Tab.Pane><Tab2 /></Tab.Pane>,
        },
        {
            menuItem: 'Вкладка 3',
            render: () => <Tab.Pane><Tab3 /></Tab.Pane>,
        },
    ];

    return (
        <Container fluid className="page-container">
            <Header as="h1" textAlign="center">
                <Icon name="shipping fast" />
                Система управления лизинговыми данными
            </Header>

            <Tab
                panes={panes}
                activeIndex={activeTab}
                onTabChange={(e, { activeIndex }) => setActiveTab(activeIndex)}
            />
        </Container>
    );
}

export default App;