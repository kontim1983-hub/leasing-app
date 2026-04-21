import React, { useState } from 'react';
import { Container, Header, Icon, Tab, Button } from 'semantic-ui-react';
import 'semantic-ui-css/semantic.min.css';
import './App.css';
import Tab1 from './components/Tab1';
import Tab2 from './components/Tab2';
import Tab3 from "./components/Tab3";
import { shutdownApp } from './utils/api';

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

    const handleShutdown = async () => {
        if (window.confirm('Вы уверены, что хотите выйти из приложения?')) {
            try {
                await shutdownApp();
                setTimeout(() => window.close(), 500);
            } catch (err) {
                console.error('Shutdown error:', err);
                window.close();
            }
        }
    };

    return (
        <Container fluid className="page-container">
            <Header as="h1" textAlign="center">
                <Icon name="shipping fast" />
                Система управления лизинговыми данными
                <Button
                    icon="sign-out"
                    color="red"
                    floated="right"
                    onClick={handleShutdown}
                    title="Выйти из приложения"
                    size="small"
                />
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