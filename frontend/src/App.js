import React, {useEffect, useState} from 'react';
import {Button, Container, Header, Icon, Image, Message, Segment, Table} from 'semantic-ui-react';
import axios from 'axios';
import 'semantic-ui-css/semantic.min.css';
import './App.css';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

function App() {
    const [records, setRecords] = useState([]);
    const [loading, setLoading] = useState(false);
    const [uploading, setUploading] = useState(false);
    const [error, setError] = useState(null);
    const [successMessage, setSuccessMessage] = useState(null);

    useEffect(() => {
        fetchRecords();
    }, []);

    const fetchRecords = async () => {
        setLoading(true);
        setError(null);
        try {
            const res = await axios.get(`${API_URL}/api/records`);
            setRecords(res.data || []);
        } catch (e) {
            setError(`Ошибка загрузки данных: ${e.message}`);
        } finally {
            setLoading(false);
        }
    };

    const handleFileUpload = async (e) => {
        const file = e.target.files && e.target.files[0];
        if (!file) return;

        if (!file.name.endsWith('.xlsx')) {
            setError('Пожалуйста, загрузите файл формата .xlsx');
            return;
        }

        setUploading(true);
        setError(null);
        setSuccessMessage(null);

        const formData = new FormData();
        formData.append('file', file);

        try {
            const res = await axios.post(`${API_URL}/api/upload`, formData, {
                headers: {'Content-Type': 'multipart/form-data'},
            });

            const processed = res.data || [];

            if (processed.length === 0) {
                setSuccessMessage('Новых изменений не обнаружено');
            } else {
                const newCount = processed.filter(r => r.is_new).length;
                const updatedCount = processed.length - newCount;

                setSuccessMessage(
                    `Обработано записей: ${processed.length} (новых: ${newCount}, обновлённых: ${updatedCount})`
                );

                fetchRecords();
            }
        } catch (e) {
            setError(`Ошибка загрузки файла: ${e.response?.data || e.message}`);
        } finally {
            setUploading(false);
            e.target.value = '';
        }
    };

    const isColumnChanged = (record, column) =>
        record.changed_columns && record.changed_columns.includes(column);

    const getCellClass = (record, column) => {
        if (record.is_new) return 'new-row';
        if (isColumnChanged(record, column)) return 'changed-cell';
        return '';
    };

    return (
        <Container fluid className="page-container">
            <Header as="h1" textAlign="center">
                <Icon name="shipping fast"/>
                Система управления лизинговыми данными
            </Header>

            <Segment>
                <Header as="h3">Загрузка Excel</Header>

                <input
                    type="file"
                    accept=".xlsx"
                    onChange={handleFileUpload}
                    disabled={uploading}
                    id="file-upload"
                    style={{display: 'none'}}
                />

                <label htmlFor="file-upload">
                    <Button
                        as="span"
                        primary
                        loading={uploading}
                        disabled={uploading}
                        icon
                        labelPosition="left"
                    >
                        <Icon name="upload"/>
                        {uploading ? 'Загрузка...' : 'Выбрать файл'}
                    </Button>
                </label>

                {error && (
                    <Message negative>
                        <Message.Header>Ошибка</Message.Header>
                        {error}
                    </Message>
                )}

                {successMessage && (
                    <Message positive>
                        <Message.Header>Успешно</Message.Header>
                        {successMessage}
                    </Message>
                )}
            </Segment>

            <Segment>
                <Header as="h3">
                    Таблица данных
                    <Button
                        floated="right"
                        size="small"
                        icon
                        labelPosition="left"
                        onClick={fetchRecords}
                        loading={loading}
                    >
                        <Icon name="refresh"/>
                        Обновить
                    </Button>
                </Header>

                {records.length === 0 ? (
                    <Message info>
                        <Message.Header>Нет данных</Message.Header>
                        Загрузите Excel файл
                    </Message>
                ) : (
                    <div className="table-wrapper">
                        <Table celled striped className="full-width-table">
                            <Table.Header>
                                <Table.Row>
                                    <Table.HeaderCell>Предмет</Table.HeaderCell>
                                    <Table.HeaderCell>Тип предмета</Table.HeaderCell>
                                    <Table.HeaderCell>Тип ТС</Table.HeaderCell>
                                    <Table.HeaderCell>VIN</Table.HeaderCell>
                                    <Table.HeaderCell>Год</Table.HeaderCell>
                                    <Table.HeaderCell>Пробег</Table.HeaderCell>
                                    <Table.HeaderCell>Дней в продаже</Table.HeaderCell>
                                    <Table.HeaderCell>Цена</Table.HeaderCell>
                                    <Table.HeaderCell>Статус</Table.HeaderCell>
                                    <Table.HeaderCell>Фото</Table.HeaderCell>
                                </Table.Row>
                            </Table.Header>

                            <Table.Body>
                                {records.map(r => (
                                    <Table.Row
                                        key={r.id}
                                        className={r.is_new ? 'new-row' : ''}
                                    >
                                        <Table.Cell className={getCellClass(r, 'subject')}>
                                            {r.subject}
                                        </Table.Cell>
                                        <Table.Cell className={getCellClass(r, 'subject_type')}>
                                            {r.subject_type}
                                        </Table.Cell>
                                        <Table.Cell className={getCellClass(r, 'vehicle_type')}>
                                            {r.vehicle_type}
                                        </Table.Cell>
                                        <Table.Cell>
                                            <strong>{r.vin}</strong>
                                        </Table.Cell>
                                        <Table.Cell className={getCellClass(r, 'year')}>
                                            {r.year}
                                        </Table.Cell>
                                        <Table.Cell className={getCellClass(r, 'mileage')}>
                                            {r.mileage}
                                        </Table.Cell>
                                        <Table.Cell className={getCellClass(r, 'days_on_sale')}>
                                            {r.days_on_sale}
                                        </Table.Cell>
                                        <Table.Cell className={getCellClass(r, 'approved_price')}>
                                            {r.old_price && r.old_price !== r.approved_price && (
                                                <span style={{color: 'red', marginRight: '5px'}}>{r.old_price}</span>
                                            )}
                                            <span style={{color: 'green'}}>{r.approved_price}</span>
                                        </Table.Cell>
                                        <Table.Cell className={getCellClass(r, 'status')}>
                                            {r.status}
                                        </Table.Cell>
                                        <Table.Cell>
                                            {r.photos && r.photos.length > 0 ? (
                                                <Image.Group size="tiny">
                                                    {r.photos.map((p, i) => (
                                                        <Image key={i} src={p}/>
                                                    ))}
                                                </Image.Group>
                                            ) : (
                                                <span className="muted">Нет фото</span>
                                            )}
                                        </Table.Cell>
                                    </Table.Row>
                                ))}
                            </Table.Body>
                        </Table>
                    </div>
                )}
            </Segment>
        </Container>
    );
}

export default App;
