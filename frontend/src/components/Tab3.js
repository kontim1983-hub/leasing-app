import React, { useEffect, useState } from 'react';
import { Button, Header, Icon, Image, Message, Segment, Table } from 'semantic-ui-react';
import {
    fetchRecordsV3,
    fetchFilesV3,
    uploadFileV3,
    clearChangedColumnsV3,
    deleteAllRecordsV3,
    exportExcelV3,
    getCellClass
} from '../utils/api';

function Tab2() {

    const [records, setRecords] = useState([]);
    const [loading, setLoading] = useState(false);
    const [uploading, setUploading] = useState(false);
    const [error, setError] = useState(null);
    const [successMessage, setSuccessMessage] = useState(null);
    const [files, setFiles] = useState([]);

    useEffect(() => {
        loadData();
    }, []);

    const loadData = async () => {
        setLoading(true);
        setError(null);
        try {
            const [recordsData, filesData] = await Promise.all([
                fetchRecordsV3(),
                fetchFilesV3()
            ]);
            setRecords(recordsData);
            setFiles(filesData);
        } catch (e) {
            setError(`Ошибка загрузки данных: ${e.message}`);
        } finally {
            setLoading(false);
        }
    };

    const handleFileUpload = async (e) => {
        const file = e.target.files?.[0];
        if (!file) return;

        if (!file.name.endsWith('.xlsx')) {
            setError('Пожалуйста, загрузите файл формата .xlsx');
            return;
        }

        setUploading(true);
        setError(null);
        setSuccessMessage(null);

        try {
            const response = await uploadFileV3(file);
            let processed = response?.data ?? response;

            if (!Array.isArray(processed)) {
                if (processed && Array.isArray(processed.records)) {
                    processed = processed.records;
                } else if (processed && Array.isArray(processed.processed)) {
                    processed = processed.processed;
                } else {
                    setSuccessMessage('Файл успешно загружен и обработан');
                    loadData();
                    return;
                }
            }
            if (processed.length === 0) {
                setSuccessMessage('Новых изменений не обнаружено');
            } else {
                const newCount = processed.filter(r => r.is_new).length;
                const updatedCount = processed.length - newCount;
                setSuccessMessage(
                    `Обработано записей: ${processed.length} (новых: ${newCount}, обновлённых: ${updatedCount})`
                );
                loadData();
            }
        } catch (err) {
            console.error('Ошибка при загрузке файла:', err);
            const message =
                err.response?.data?.message ||
                err.response?.data ||
                err.message ||
                'Неизвестная ошибка';
            setError(`Ошибка загрузки файла: ${message}`);
        } finally {
            setUploading(false);
            e.target.value = '';
        }
    };

    const handleClearChangedColumns = async () => {
        if (!window.confirm("Вы уверены, что хотите очистить все значения в колонке ChangedColumns?")) return;

        try {
            const data = await clearChangedColumnsV3();
            alert(data.message || "ChangedColumns очищены!");
            loadData();
        } catch (err) {
            alert("Ошибка: " + err.message);
        }
    };

    const handleDeleteDatabase = async () => {
        if (!window.confirm("ВНИМАНИЕ! Это удалит ВСЕ данные из базы навсегда. Вы уверены?")) return;

        const secondConfirm = window.prompt("Для подтверждения введите слово: УДАЛИТЬ");
        if (secondConfirm !== "УДАЛИТЬ") {
            alert("Операция отменена");
            return;
        }

        try {
            const data = await deleteAllRecordsV3();
            alert(data.message || "Все записи успешно удалены!");
            setRecords([]);
            setFiles([]);
            setSuccessMessage("База данных очищена");
        } catch (err) {
            console.error(err);
            alert("Ошибка при удалении: " + err.message);
        }
    };

    const handleExportExcel = async () => {
        try {
            await exportExcelV3();
        } catch (err) {
            alert("Ошибка экспорта: " + err.message);
        }
    };

    return (
        <>
            <Segment>
                <Header as="h3">Загрузка Excel</Header>
                <input
                    type="file"
                    accept=".xlsx"
                    onChange={handleFileUpload}
                    disabled={uploading}
                    id="file-upload-V3"
                    style={{ display: 'none' }}
                />
                <label htmlFor="file-upload-V3">
                    <Button as="span" primary loading={uploading} disabled={uploading} icon labelPosition="left">
                        <Icon name="upload" />
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

                {files.length > 0 && (
                    <div style={{ marginBottom: 16 }}>
                        <Header as="h4">Загруженные файлы:</Header>
                        <ul>
                            {files.map(f => (
                                <li key={f}>{f}</li>
                            ))}
                        </ul>
                    </div>
                )}
            </Segment>

            <div style={{ margin: '20px', padding: '10px', border: '1px solid red' }}>
                <h3>Админские действия</h3>
                <button onClick={handleClearChangedColumns} style={{ marginRight: '10px', padding: '10px', background: '#ff9800' }}>
                    Очистить ChangedColumns
                </button>
                <button onClick={handleDeleteDatabase} style={{ marginRight: '10px', padding: '10px', background: '#f44336', color: 'white' }}>
                    Удалить всю базу
                </button>
                <button onClick={handleExportExcel} style={{ padding: '10px', background: '#4caf50', color: 'white' }}>
                    Экспорт в Excel
                </button>
            </div>

            <Segment>
                <Header as="h3">
                    Таблица данных
                    <Button floated="right" size="small" icon labelPosition="left" onClick={loadData} loading={loading}>
                        <Icon name="refresh" />
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
                                    <Table.HeaderCell>Марка</Table.HeaderCell>
                                    <Table.HeaderCell>Модель</Table.HeaderCell>
                                    <Table.HeaderCell>VIN</Table.HeaderCell>
                                    <Table.HeaderCell>Срок экспозиции(дн.)</Table.HeaderCell>
                                    <Table.HeaderCell>Вид ТС</Table.HeaderCell>
                                    <Table.HeaderCell>Подвид ТС</Table.HeaderCell>
                                    <Table.HeaderCell>Год выпуска</Table.HeaderCell>
                                    <Table.HeaderCell>Пробег</Table.HeaderCell>
                                    <Table.HeaderCell>Город</Table.HeaderCell>
                                    <Table.HeaderCell>Актуальная стоимость</Table.HeaderCell>
                                    <Table.HeaderCell>Старая стоимость</Table.HeaderCell>
                                    <Table.HeaderCell>Разница</Table.HeaderCell>
                                    <Table.HeaderCell>Статус</Table.HeaderCell>
                                    <Table.HeaderCell>Фото</Table.HeaderCell>
                                </Table.Row>
                            </Table.Header>

                            <Table.Body>
                                {records.map(r => (
                                    <Table.Row key={r.id} className={r.is_new ? 'new-row' : ''}>
                                        <Table.Cell className={getCellClass(r, 'brand')}>{r.brand}</Table.Cell>
                                        <Table.Cell className={getCellClass(r, 'model')}>{r.model}</Table.Cell>
                                        <Table.Cell><strong>{r.vin}</strong></Table.Cell>
                                        <Table.Cell className={getCellClass(r, 'exposure_period')}>{r.exposure_period}</Table.Cell>
                                        <Table.Cell className={getCellClass(r, 'vehicle_type')}>{r.vehicle_type}</Table.Cell>
                                        <Table.Cell className={getCellClass(r, 'vehicle_subtype')}>{r.vehicle_subtype}</Table.Cell>
                                        <Table.Cell className={getCellClass(r, 'year')}>{r.year}</Table.Cell>
                                        <Table.Cell className={getCellClass(r, 'mileage')}>{r.mileage}</Table.Cell>
                                        <Table.Cell className={getCellClass(r, 'city')}>{r.city}</Table.Cell>
                                        <Table.Cell className={getCellClass(r, 'actual_price')}>
                                            {r.old_price && r.old_price !== r.actual_price && (
                                                <span style={{ color: 'red', marginRight: '5px' }}>{r.old_price}</span>
                                            )}
                                            <span style={{ color: 'green' }}>{r.actual_price}</span>
                                        </Table.Cell>
                                        <Table.Cell>
                                            {r.old_price && r.old_price !== r.actual_price && (
                                                <span style={{ color: 'red' }}>{r.old_price}</span>
                                            )}
                                        </Table.Cell>
                                        <Table.Cell>
                                            <span style={{ color: 'green' }}>
                                                {parseFloat((r.old_price || "0").replace(/,/g, "")) - parseFloat((r.actual_price || "0").replace(/,/g, ""))}
                                            </span>
                                        </Table.Cell>
                                        <Table.Cell className={getCellClass(r, 'status')}>{r.status}</Table.Cell>
                                        <Table.Cell>
                                            {r.photos?.length > 0 ? (
                                                <Image.Group size="tiny">
                                                    {r.photos.map((p, i) => (
                                                        <Image key={i} src={p} />
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
        </>
    );
}

export default Tab2;